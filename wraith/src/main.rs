// SPDX-License-Identifier: AGPL-3.0-only
use std::env;
use std::error::Error;
use std::io::{Read, Write};
use std::net::{IpAddr, TcpStream};
use std::sync::Arc;

use aes_gcm::aead::{Aead, KeyInit};
use aes_gcm::{Aes256Gcm, Nonce};
use hkdf::Hkdf;
use rand::RngCore;
use rustls::pki_types::{ServerName, UnixTime};
use rustls::{
    ClientConfig, ClientConnection, DigitallySignedStruct, SignatureScheme, Stream,
};
use serde_json::{json, Value};
use sha2::Sha256;
use x25519_dalek::{PublicKey, StaticSecret};

const SALT: &[u8] = b"sunder-whisper-v1";
const INFO: &[u8] = b"session-key";

fn main() -> Result<(), Box<dyn Error>> {
    let mut url = "https://localhost:8443".to_string();
    let mut loop_mode = false;
    let mut interval: u64 = 5;
    let mut args = env::args().skip(1);
    while let Some(arg) = args.next() {
        match arg.as_str() {
            "--version" | "-V" => {
                println!("wraith {}", env!("CARGO_PKG_VERSION"));
                return Ok(());
            }
            "--loop" => loop_mode = true,
            "--interval" => {
                if let Some(v) = args.next() {
                    if let Ok(n) = v.parse::<u64>() {
                        interval = n;
                    }
                }
            }
            _ if arg.starts_with("https://") => url = arg,
            _ => {}
        }
    }
    let (host, port) = parse_url(&url)?;

    let (status, body) = https_request(&host, port, "GET", "/whisper/v1/handshake/start", "")?;
    check_status(status)?;
    let start: Value = serde_json::from_str(&body)?;
    let session = start["session"].as_str().unwrap().to_string();
    let server_eph_hex = start["eph_pub"].as_str().unwrap();

    let client_secret = StaticSecret::random_from_rng(rand::rngs::OsRng);
    let client_pub = PublicKey::from(&client_secret);
    let server_pub_bytes: [u8; 32] = from_hex(server_eph_hex)?.try_into().map_err(|_| "bad server key")?;
    let shared = client_secret.diffie_hellman(&PublicKey::from(server_pub_bytes));
    let key = derive_key(shared.as_bytes())?;

    let hello = json!({
        "id": random_hex(8),
        "hostname": hostname(),
        "os": env::consts::OS,
        "arch": env::consts::ARCH,
        "user": username(),
    });
    let (nonce, ct) = seal(&key, &hello.to_string())?;
    let complete = json!({
        "session": session,
        "eph_pub": to_hex(client_pub.as_bytes()),
        "nonce": nonce,
        "ct": ct,
    });
    let (status, body) = https_request(&host, port, "POST", "/whisper/v1/handshake/complete", &complete.to_string())?;
    check_status(status)?;
    let env: Value = serde_json::from_str(&body)?;
    let ack_plain = open(
        &key,
        env["nonce"].as_str().unwrap(),
        env["ct"].as_str().unwrap(),
    )?;
    let ack: Value = serde_json::from_str(&ack_plain)?;
    let shard_id = ack["shard_id"].as_str().unwrap().to_string();
    println!("whisper session established with {}", ack["server"]);
    println!(
        "shard_id={} session_id={} cadence_s={}",
        shard_id, ack["session_id"], ack["cadence_s"]
    );

    if loop_mode {
        loop {
            let res = send_word(&key, &shard_id, &host, port, "breath")?;
            println!("word=breath result={} ok={}", res["result"], res["ok"]);
            std::thread::sleep(std::time::Duration::from_secs(interval));
        }
    }
    let res = send_word(&key, &shard_id, &host, port, "breath")?;
    println!("word=breath result={} ok={}", res["result"], res["ok"]);
    Ok(())
}

fn send_word(
    key: &[u8; 32],
    shard_id: &str,
    host: &str,
    port: u16,
    word: &str,
) -> Result<Value, Box<dyn Error>> {
    let payload = json!({"shard_id": shard_id, "word": word});
    let (nonce, ct) = seal(key, &payload.to_string())?;
    let req = json!({"shard_id": shard_id, "envelope": {"nonce": nonce, "ct": ct}});
    let (status, body) = https_request(host, port, "POST", "/whisper/v1/word", &req.to_string())?;
    check_status(status)?;
    let env: Value = serde_json::from_str(&body)?;
    let plain = open(key, env["nonce"].as_str().unwrap(), env["ct"].as_str().unwrap())?;
    Ok(serde_json::from_str(&plain)?)
}

fn parse_url(url: &str) -> Result<(String, u16), Box<dyn Error>> {
    let rest = url
        .strip_prefix("https://")
        .ok_or("wraith: only https urls are supported")?;
    let (host, port) = rest
        .split_once(':')
        .ok_or("wraith: url must include a port")?;
    let port: u16 = port.parse()?;
    Ok((host.to_string(), port))
}

fn check_status(status: u16) -> Result<(), Box<dyn Error>> {
    if status < 200 || status > 299 {
        return Err(format!("wraith: server returned HTTP {status}").into());
    }
    Ok(())
}

#[derive(Debug)]
struct NoVerify(Arc<rustls::crypto::CryptoProvider>);

impl rustls::client::danger::ServerCertVerifier for NoVerify {
    fn verify_server_cert(
        &self,
        _end_entity: &rustls::pki_types::CertificateDer,
        _intermediates: &[rustls::pki_types::CertificateDer],
        _server_name: &ServerName,
        _ocsp: &[u8],
        _now: UnixTime,
    ) -> Result<rustls::client::danger::ServerCertVerified, rustls::Error> {
        Ok(rustls::client::danger::ServerCertVerified::assertion())
    }

    fn verify_tls12_signature(
        &self,
        message: &[u8],
        cert: &rustls::pki_types::CertificateDer,
        dss: &DigitallySignedStruct,
    ) -> Result<rustls::client::danger::HandshakeSignatureValid, rustls::Error> {
        rustls::crypto::verify_tls12_signature(
            message,
            cert,
            dss,
            &self.0.signature_verification_algorithms,
        )
    }

    fn verify_tls13_signature(
        &self,
        message: &[u8],
        cert: &rustls::pki_types::CertificateDer,
        dss: &DigitallySignedStruct,
    ) -> Result<rustls::client::danger::HandshakeSignatureValid, rustls::Error> {
        rustls::crypto::verify_tls13_signature(
            message,
            cert,
            dss,
            &self.0.signature_verification_algorithms,
        )
    }

    fn supported_verify_schemes(&self) -> Vec<SignatureScheme> {
        self.0.signature_verification_algorithms.supported_schemes()
    }
}

fn https_request(
    host: &str,
    port: u16,
    method: &str,
    path: &str,
    body: &str,
) -> Result<(u16, String), Box<dyn Error>> {
    let provider = Arc::new(rustls::crypto::ring::default_provider());
    let config = ClientConfig::builder_with_provider(provider.clone())
        .with_protocol_versions(&[&rustls::version::TLS13, &rustls::version::TLS12])?
        .dangerous()
        .with_custom_certificate_verifier(Arc::new(NoVerify(provider.clone())))
        .with_no_client_auth();
    let server_name = match host.parse::<IpAddr>() {
        Ok(ip) => ServerName::IpAddress(ip.into()),
        Err(_) => ServerName::try_from(host.to_string())?,
    };
    let mut sock = TcpStream::connect((host, port))?;
    let mut conn = ClientConnection::new(Arc::new(config), server_name)?;
    let mut tls = Stream::new(&mut conn, &mut sock);
    let request = format!(
        "{method} {path} HTTP/1.1\r\n\
         Host: {host}:{port}\r\n\
         Content-Type: application/json\r\n\
         Content-Length: {}\r\n\
         Connection: close\r\n\
         \r\n\
         {body}",
        body.len()
    );
    tls.write_all(request.as_bytes())?;
    let mut raw = Vec::new();
    tls.read_to_end(&mut raw)?;
    let text = String::from_utf8_lossy(&raw).into_owned();
    let status: u16 = text
        .split_whitespace()
        .nth(1)
        .ok_or("wraith: malformed response")?
        .parse()?;
    let payload = text.split("\r\n\r\n").nth(1).unwrap_or("").to_string();
    Ok((status, payload))
}

fn derive_key(shared: &[u8]) -> Result<[u8; 32], Box<dyn Error>> {
    let hk = Hkdf::<Sha256>::new(Some(SALT), shared);
    let mut okm = [0u8; 32];
    hk.expand(INFO, &mut okm)
        .map_err(|e| format!("wraith: key expansion failed: {e:?}"))?;
    Ok(okm)
}

fn seal(key: &[u8; 32], plain: &str) -> Result<(String, String), Box<dyn Error>> {
    let cipher = Aes256Gcm::new_from_slice(key)?;
    let mut nonce_bytes = [0u8; 12];
    rand::rngs::OsRng.fill_bytes(&mut nonce_bytes);
    let nonce = Nonce::from_slice(&nonce_bytes);
    let ct = cipher.encrypt(nonce, plain.as_bytes()).map_err(|_| "wraith: seal failed")?;
    Ok((to_hex(&nonce_bytes), to_hex(&ct)))
}

fn open(key: &[u8; 32], nonce_hex: &str, ct_hex: &str) -> Result<String, Box<dyn Error>> {
    let cipher = Aes256Gcm::new_from_slice(key)?;
    let nonce_bytes: [u8; 12] = from_hex(nonce_hex)?.try_into().map_err(|_| "bad nonce")?;
    let nonce = Nonce::from_slice(&nonce_bytes);
    let ct = from_hex(ct_hex)?;
    let plain = cipher.decrypt(nonce, ct.as_ref()).map_err(|_| "wraith: open failed")?;
    Ok(String::from_utf8(plain)?)
}

fn to_hex(bytes: &[u8]) -> String {
    let mut out = String::with_capacity(bytes.len() * 2);
    for b in bytes {
        out.push_str(&format!("{b:02x}"));
    }
    out
}

fn from_hex(s: &str) -> Result<Vec<u8>, Box<dyn Error>> {
    let mut out = Vec::with_capacity(s.len() / 2);
    let b = s.as_bytes();
    let mut i = 0;
    while i + 1 < b.len() {
        let hi = hex_val(b[i])?;
        let lo = hex_val(b[i + 1])?;
        out.push((hi << 4) | lo);
        i += 2;
    }
    Ok(out)
}

fn hex_val(c: u8) -> Result<u8, Box<dyn Error>> {
    match c {
        b'0'..=b'9' => Ok(c - b'0'),
        b'a'..=b'f' => Ok(c - b'a' + 10),
        _ => Err("bad hex".into()),
    }
}

fn random_hex(n: usize) -> String {
    let mut bytes = vec![0u8; n];
    rand::rngs::OsRng.fill_bytes(&mut bytes);
    to_hex(&bytes)
}

fn hostname() -> String {
    env::var("HOSTNAME")
        .or_else(|_| env::var("COMPUTERNAME"))
        .unwrap_or_else(|_| "unknown".to_string())
}

fn username() -> String {
    env::var("USER")
        .or_else(|_| env::var("USERNAME"))
        .unwrap_or_else(|_| "unknown".to_string())
}
