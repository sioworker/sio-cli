use serde::{Deserialize, Deserializer, Serialize};
use std::{collections::BTreeMap, fs, io, path::PathBuf};

#[derive(Serialize, Deserialize, Clone, Default)]
pub struct Host {
	#[serde(default)]
	pub url: String,
	#[serde(default, deserialize_with = "nstr")]
	pub token: String,
	#[serde(default, deserialize_with = "nstr", skip_serializing_if = "String::is_empty")]
	pub session: String, // web sessionid from sio config hosts login
}

#[derive(Serialize, Deserialize, Default)]
pub struct Cfg {
	#[serde(default, deserialize_with = "nstr")]
	pub main: String,
	#[serde(default, deserialize_with = "nstr", skip_serializing_if = "String::is_empty")]
	pub lang: String,
	#[serde(rename = "display_quotes", default, skip_serializing_if = "Option::is_none")]
	pub quotes: Option<bool>,
	#[serde(default, deserialize_with = "nmap")]
	pub hosts: BTreeMap<String, Host>, // btree so it saves sorted
}

pub fn nstr<'de, D: Deserializer<'de>>(d: D) -> Result<String, D::Error> {
	Ok(Option::<String>::deserialize(d)?.unwrap_or_default()) // null -> ""
}

fn nmap<'de, D: Deserializer<'de>>(d: D) -> Result<BTreeMap<String, Host>, D::Error> {
	Ok(Option::<BTreeMap<String, Host>>::deserialize(d)?.unwrap_or_default())
}

pub fn cfg_path() -> PathBuf {
	let d = std::env::var_os("XDG_CONFIG_HOME").filter(|v| !v.is_empty()).map(PathBuf::from).unwrap_or_else(|| dirs::home_dir().unwrap_or_default().join(".config"));
	d.join("sio").join("cfg.json")
}

impl Cfg {
	pub fn load() -> Cfg {
		fs::read(cfg_path()).ok().and_then(|b| serde_json::from_slice(&b).ok()).unwrap_or_default()
	}

	pub fn save(&self) -> io::Result<()> {
		let p = cfg_path();
		mkdir(p.parent().unwrap())?;
		let mut b = Vec::new();
		let mut s = serde_json::Serializer::with_formatter(&mut b, serde_json::ser::PrettyFormatter::with_indent(b"\t"));
		self.serialize(&mut s).map_err(io::Error::other)?;
		write(&p, &b) // has tokens
	}

	pub fn quotes_on(&self) -> bool {
		self.quotes.unwrap_or(true) // def on
	}
}

pub fn mkdir(p: &std::path::Path) -> io::Result<()> {
	let mut d = fs::DirBuilder::new();
	d.recursive(true);
	#[cfg(unix)]
	std::os::unix::fs::DirBuilderExt::mode(&mut d, 0o700);
	d.create(p)
}

pub fn write(p: &std::path::Path, b: &[u8]) -> io::Result<()> {
	let mut o = fs::OpenOptions::new(); // 0600, only applies on create
	o.write(true).create(true).truncate(true);
	#[cfg(unix)]
	std::os::unix::fs::OpenOptionsExt::mode(&mut o, 0o600);
	io::Write::write_all(&mut o.open(p)?, b)
}
