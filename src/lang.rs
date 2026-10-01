use rust_embed::Embed;
use std::{
	collections::HashMap,
	fs,
	path::PathBuf,
	sync::{LazyLock, RwLock},
};

#[derive(Embed)]
#[folder = "lang/"]
struct Files;

static L: LazyLock<RwLock<HashMap<String, String>>> = LazyLock::new(Default::default);

#[macro_export]
macro_rules! t {
	($k:expr) => { $crate::lang::t($k, &[]) };
	($k:expr, $($a:expr),+) => { $crate::lang::t($k, &[$(format!("{}", $a)),+]) };
}

fn user_lang() -> PathBuf {
	crate::cfg::cfg_path().parent().unwrap().join("lang")
}

pub fn code(saved: &str) -> String {
	if let Ok(v) = std::env::var("SIO_LANG") // SIO_LANG > sio config lang > system > en
		&& !v.is_empty()
	{
		return v;
	}
	if !saved.is_empty() {
		return saved.into();
	}
	for k in ["LC_ALL", "LC_MESSAGES", "LANG"] {
		if let Ok(v) = std::env::var(k)
			&& !v.is_empty()
		{
			let v = v.split('.').next().unwrap().split('_').next().unwrap();
			if v == "C" || v == "POSIX" {
				return "en".into();
			}
			return v.to_lowercase();
		}
	}
	"en".into()
}

pub fn jsonc(b: &[u8]) -> Vec<u8> {
	let (mut o, mut i, n, mut st) = (Vec::with_capacity(b.len()), 0, b.len(), false); // drop // and /* */ comments, then trailing commas
	while i < n {
		let mut c = b[i];
		if st {
			if c == b'\\' && i + 1 < n {
				o.push(c);
				i += 1;
				c = b[i];
			} else if c == b'"' {
				st = false;
			}
		} else if c == b'"' {
			st = true;
		} else if c == b'/' && i + 1 < n && b[i + 1] == b'/' {
			while i < n && b[i] != b'\n' {
				i += 1;
			}
			continue;
		} else if c == b'/' && i + 1 < n && b[i + 1] == b'*' {
			i += 2;
			while i + 1 < n && !(b[i] == b'*' && b[i + 1] == b'/') {
				i += 1;
			}
			i += 2;
			continue;
		}
		o.push(c);
		i += 1;
	}
	let (b, mut o, mut i, mut st) = (o, Vec::new(), 0, false);
	let n = b.len();
	while i < n {
		let mut c = b[i];
		if st {
			if c == b'\\' && i + 1 < n {
				o.push(c);
				i += 1;
				c = b[i];
			} else if c == b'"' {
				st = false;
			}
		} else if c == b'"' {
			st = true;
		} else if c == b',' {
			let mut j = i + 1;
			while j < n && matches!(b[j], b' ' | b'\t' | b'\r' | b'\n') {
				j += 1;
			}
			if j < n && (b[j] == b'}' || b[j] == b']') {
				i += 1;
				continue;
			}
		}
		o.push(c);
		i += 1;
	}
	o
}

fn parse(b: &[u8]) -> HashMap<String, String> {
	serde_json::from_slice::<HashMap<String, serde_json::Value>>(&jsonc(b)).map(|m| m.into_iter().filter_map(|(k, v)| v.as_str().map(|s| (k, s.to_string()))).collect()).unwrap_or_default() // non string vals skipped
}

fn builtin(code: &str) -> Option<Vec<u8>> {
	Files::get(&format!("{code}.jsonc")).map(|f| f.data.into_owned())
}

fn user(code: &str) -> Option<Vec<u8>> {
	fs::read(user_lang().join(format!("{code}.jsonc"))).ok()
}

pub fn load(code: &str) {
	let mut l = L.write().unwrap(); // en -> built-in code -> ~/.config/sio/lang/code.jsonc
	l.clear();
	for b in [builtin("en"), builtin(code), user(code)].into_iter().flatten() {
		l.extend(parse(&b));
	}
}

pub fn langs() -> Vec<String> {
	let mut o: Vec<String> = Files::iter().filter_map(|f| f.strip_suffix(".jsonc").map(String::from)).collect(); // built-in + ~/.config/sio/lang/
	if let Ok(rd) = fs::read_dir(user_lang()) {
		o.extend(rd.flatten().filter_map(|e| e.file_name().to_str().and_then(|n| n.strip_suffix(".jsonc")).map(String::from)));
	}
	o.sort();
	o.dedup();
	o
}

pub fn name(code: &str) -> String {
	let mut m = HashMap::new();
	for b in [builtin(code), user(code)].into_iter().flatten() {
		m.extend(parse(&b));
	}
	m.remove("lang_name").unwrap_or_default()
}

pub fn t(k: &str, a: &[String]) -> String {
	let l = L.read().unwrap(); // %s filled in order, %% = %
	let s = l.get(k).map(String::as_str).unwrap_or(k);
	if a.is_empty() {
		return s.into();
	}
	let (mut o, mut it, mut a) = (String::with_capacity(s.len()), s.chars().peekable(), a.iter());
	while let Some(c) = it.next() {
		if c == '%' {
			match it.peek() {
				Some('s') | Some('d') | Some('v') => {
					it.next();
					o += a.next().map(String::as_str).unwrap_or("");
					continue;
				}
				Some('%') => {
					it.next();
				}
				_ => {}
			}
		}
		o.push(c);
	}
	o
}

#[cfg(test)]
mod tests {
	use super::*;

	fn verbs(s: &str) -> usize {
		s.matches('%').count() - 2 * s.matches("%%").count()
	}

	#[test]
	fn jsonc_strips() {
		let i = "// top\n{\n\t\"a\": \"x // not a comment\", /* block */\n\t\"b\": \"q\\\"/*\", // tricky\n\t\"c\": [1, 2,],\n}";
		let m: serde_json::Value = serde_json::from_slice(&jsonc(i.as_bytes())).unwrap_or_else(|e| panic!("{e}: {}", String::from_utf8_lossy(&jsonc(i.as_bytes()))));
		assert_eq!(m["a"], "x // not a comment");
		assert_eq!(m["b"], "q\"/*");
		assert_eq!(m["c"].as_array().unwrap().len(), 2);
	}

	#[test]
	fn lang_files() {
		let en = parse(&builtin("en").unwrap());
		assert!(!en.is_empty(), "en.jsonc didnt parse");
		let mut bad = vec![];
		for f in Files::iter() {
			let b = Files::get(&f).unwrap().data;
			let m: HashMap<String, String> = match serde_json::from_slice(&jsonc(&b)) {
				Ok(m) => m,
				Err(e) => {
					bad.push(format!("{f}: {e}"));
					continue;
				}
			};
			for (k, v) in &en {
				match m.get(k) {
					None => bad.push(format!("{f}: missing {k}")),
					Some(s) if verbs(s) != verbs(v) => bad.push(format!("{f}: {k} has {} args, en has {}", verbs(s), verbs(v))),
					_ => {}
				}
			}
			for k in m.keys().filter(|k| !en.contains_key(*k)) {
				bad.push(format!("{f}: {k} not in en.jsonc"));
			}
		}
		assert!(bad.is_empty(), "\n{}", bad.join("\n"));
	}

	#[test]
	fn fmt_args() {
		L.write().unwrap().insert("x".into(), "%s is %s, 100%%".into());
		assert_eq!(t("x", &["a".into(), "b".into()]), "a is b, 100%");
	}
}
