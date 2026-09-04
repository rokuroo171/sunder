// SPDX-License-Identifier: AGPL-3.0-only
use std::env;
use std::fs;
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::{Duration, Instant};
#[cfg(target_os = "linux")]
use std::time::{SystemTime, UNIX_EPOCH};

use serde_json::Value;

// Outcome is the answer a Word produces on this machine
pub struct Outcome {
    pub ok: bool,
    pub output: String,
}

// run executes one Word and returns its outcome
pub fn run(word: &str, args: &Value) -> Outcome {
    match word {
        "utter" => utter(args),
        "gaze" => gaze(args),
        "unfold" => unfold(args),
        "pulse" => pulse(),
        "anatomy" => anatomy(),
        _ => Outcome {
            ok: false,
            output: format!("unknown word: {word}"),
        },
    }
}

pub fn hostname() -> String {
    env::var("HOSTNAME")
        .or_else(|_| env::var("COMPUTERNAME"))
        .unwrap_or_else(|_| "unknown".to_string())
}

pub fn username() -> String {
    env::var("USER")
        .or_else(|_| env::var("USERNAME"))
        .unwrap_or_else(|_| "unknown".to_string())
}

fn utter(args: &Value) -> Outcome {
    let command = match args.get("command").and_then(Value::as_str) {
        Some(c) => c,
        None => return Outcome { ok: false, output: "utter needs a command".to_string() },
    };
    let timeout = args
        .get("t")
        .and_then(Value::as_str)
        .and_then(|s| s.parse::<u64>().ok())
        .unwrap_or(20);
    let mut cmd = shell(command);
    capture(&mut cmd, timeout)
}

#[cfg(windows)]
fn shell(command: &str) -> Command {
    let mut c = Command::new("cmd");
    c.args(["/C", command]);
    c
}

#[cfg(not(windows))]
fn shell(command: &str) -> Command {
    let mut c = Command::new("sh");
    c.args(["-c", command]);
    c
}

// capture runs a command to completion with a timeout and returns its outcome
fn capture(cmd: &mut Command, timeout_secs: u64) -> Outcome {
    let out_path = temp_path("out");
    let err_path = temp_path("err");
    let out_file = match fs::File::create(&out_path) {
        Ok(f) => f,
        Err(e) => return Outcome { ok: false, output: format!("capture: {e}") },
    };
    let err_file = match fs::File::create(&err_path) {
        Ok(f) => f,
        Err(e) => return Outcome { ok: false, output: format!("capture: {e}") },
    };
    let mut child = match cmd
        .stdin(Stdio::null())
        .stdout(Stdio::from(out_file))
        .stderr(Stdio::from(err_file))
        .spawn()
    {
        Ok(c) => c,
        Err(e) => {
            let _ = fs::remove_file(&out_path);
            let _ = fs::remove_file(&err_path);
            return Outcome { ok: false, output: format!("spawn failed: {e}") };
        }
    };
    let deadline = Instant::now() + Duration::from_secs(timeout_secs);
    let status = loop {
        match child.try_wait() {
            Ok(Some(st)) => break st,
            Ok(None) => {
                if Instant::now() >= deadline {
                    let _ = child.kill();
                    let _ = child.wait();
                    let outcome = timed_out(&out_path, &err_path, timeout_secs);
                    let _ = fs::remove_file(&out_path);
                    let _ = fs::remove_file(&err_path);
                    return outcome;
                }
                std::thread::sleep(Duration::from_millis(100));
            }
            Err(e) => {
                let _ = child.kill();
                let _ = child.wait();
                let _ = fs::remove_file(&out_path);
                let _ = fs::remove_file(&err_path);
                return Outcome { ok: false, output: format!("wait failed: {e}") };
            }
        }
    };
    let ok = status.success();
    let stdout = fs::read(&out_path).unwrap_or_default();
    let stderr = fs::read(&err_path).unwrap_or_default();
    let _ = fs::remove_file(&out_path);
    let _ = fs::remove_file(&err_path);
    let mut output = lossy(&stdout);
    if !ok {
        let tail = lossy(&stderr);
        if !output.is_empty() {
            output.push('\n');
        }
        output.push_str(&format!("exit: {status}"));
        if !tail.is_empty() {
            output.push('\n');
            output.push_str(&tail);
        }
    }
    Outcome { ok, output }
}

fn timed_out(out_path: &Path, err_path: &Path, timeout_secs: u64) -> Outcome {
    let stdout = fs::read(out_path).unwrap_or_default();
    let stderr = fs::read(err_path).unwrap_or_default();
    let mut output = lossy(&stdout);
    let tail = lossy(&stderr);
    if !output.is_empty() {
        output.push('\n');
    }
    output.push_str(&format!("timed out after {timeout_secs}s"));
    if !tail.is_empty() {
        output.push('\n');
        output.push_str(&tail);
    }
    Outcome { ok: false, output }
}

fn temp_path(kind: &str) -> PathBuf {
    static SEQ: AtomicU64 = AtomicU64::new(0);
    let seq = SEQ.fetch_add(1, Ordering::Relaxed);
    env::temp_dir().join(format!("sunder-wraith-{kind}-{}-{seq}", std::process::id()))
}

// lossy decodes bytes as text, normalizes newlines to Unix, and trims the tail
fn lossy(bytes: &[u8]) -> String {
    let s = String::from_utf8_lossy(bytes)
        .replace("\r\n", "\n")
        .replace('\r', "\n");
    s.trim_end().to_string()
}

fn gaze(args: &Value) -> Outcome {
    let path = args.get("path").and_then(Value::as_str).unwrap_or(".");
    let entries = match fs::read_dir(path) {
        Ok(rd) => rd,
        Err(e) => return Outcome { ok: false, output: format!("gaze: {e}") },
    };
    let mut names: Vec<String> = entries
        .filter_map(|e| e.ok())
        .map(|e| e.file_name().to_string_lossy().into_owned())
        .collect();
    names.sort();
    Outcome { ok: true, output: names.join("\n") }
}

fn unfold(args: &Value) -> Outcome {
    let path = match args.get("path").and_then(Value::as_str) {
        Some(p) => p,
        None => return Outcome { ok: false, output: "unfold needs a path".to_string() },
    };
    let data = match fs::read(path) {
        Ok(d) => d,
        Err(e) => return Outcome { ok: false, output: format!("unfold: {e}") },
    };
    let offset = args
        .get("o")
        .and_then(Value::as_str)
        .and_then(|s| s.parse::<usize>().ok())
        .unwrap_or(0);
    let len = args
        .get("n")
        .and_then(Value::as_str)
        .and_then(|s| s.parse::<usize>().ok())
        .unwrap_or(usize::MAX);
    let slice: Vec<u8> = data.iter().skip(offset).take(len).cloned().collect();
    let kind = args.get("t").and_then(Value::as_str).unwrap_or("text");
    let output = match kind {
        "hex" => to_hex(&slice),
        "base64" => base64(&slice),
        _ => lossy(&slice),
    };
    Outcome { ok: true, output }
}

fn to_hex(bytes: &[u8]) -> String {
    let mut out = String::with_capacity(bytes.len() * 2);
    for b in bytes {
        out.push_str(&format!("{b:02x}"));
    }
    out
}

fn base64(bytes: &[u8]) -> String {
    const TABLE: &[u8; 64] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut out = String::with_capacity(bytes.len().div_ceil(3) * 4);
    for chunk in bytes.chunks(3) {
        let b0 = chunk[0] as u32;
        let b1 = *chunk.get(1).unwrap_or(&0) as u32;
        let b2 = *chunk.get(2).unwrap_or(&0) as u32;
        let n = (b0 << 16) | (b1 << 8) | b2;
        out.push(TABLE[((n >> 18) & 63) as usize] as char);
        out.push(TABLE[((n >> 12) & 63) as usize] as char);
        out.push(if chunk.len() > 1 { TABLE[((n >> 6) & 63) as usize] as char } else { '=' });
        out.push(if chunk.len() > 2 { TABLE[(n & 63) as usize] as char } else { '=' });
    }
    out
}

fn pulse() -> Outcome {
    #[cfg(windows)]
    {
        let mut cmd = Command::new("powershell");
        cmd.args(["-NoProfile", "-Command", "Get-CimInstance Win32_Process | ForEach-Object { $line = if ($_.CommandLine) { $_.CommandLine } else { $_.Name }; $_.ProcessId.ToString() + '  ' + $line }"]);
        capture(&mut cmd, 15)
    }
    #[cfg(not(windows))]
    {
        let mut cmd = Command::new("ps");
        cmd.args(["-eo", "pid=,args="]);
        capture(&mut cmd, 10)
    }
}

fn anatomy() -> Outcome {
    let mut lines = vec![
        format!("host={}", hostname()),
        format!("os={}", env::consts::OS),
        format!("arch={}", env::consts::ARCH),
        format!("user={}", username()),
    ];
    lines.push(format!("kernel={}", kernel_version()));
    #[cfg(target_os = "linux")]
    {
        if let Some(uptime) = linux_uptime() {
            let now = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .map(|d| d.as_secs())
                .unwrap_or(0);
            lines.push(format!("boot={}", now.saturating_sub(uptime)));
        }
    }
    Outcome { ok: true, output: lines.join("\n") }
}

#[cfg(windows)]
fn kernel_version() -> String {
    let mut cmd = Command::new("cmd");
    cmd.args(["/C", "ver"]);
    let out = capture(&mut cmd, 5);
    if out.ok {
        out.output
    } else {
        "unknown".to_string()
    }
}

#[cfg(not(windows))]
fn kernel_version() -> String {
    let mut cmd = Command::new("uname");
    cmd.args(["-srm"]);
    let out = capture(&mut cmd, 5);
    if out.ok {
        out.output
    } else {
        "unknown".to_string()
    }
}

#[cfg(target_os = "linux")]
fn linux_uptime() -> Option<u64> {
    let text = fs::read_to_string("/proc/uptime").ok()?;
    let secs = text.split_whitespace().next()?.parse::<f64>().ok()?;
    Some(secs as u64)
}
