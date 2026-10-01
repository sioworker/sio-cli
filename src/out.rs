use crate::{spin, t};
use regex::Regex;
use std::{
	io::{IsTerminal, Write},
	process::{Command, Stdio},
	sync::LazyLock,
	time::Duration,
};

pub const RED: &str = "31";
pub const GRN: &str = "32";
pub const YLW: &str = "33";
pub const CYN: &str = "36";
pub const GRY: &str = "90";
pub const DIM: &str = "2";
pub const BOLD: &str = "1";

fn nocol() -> bool {
	std::env::var_os("NO_COLOR").is_some_and(|v| !v.is_empty())
}

pub static COL_OUT: LazyLock<bool> = LazyLock::new(|| !nocol() && std::io::stdout().is_terminal());
pub static COL_ERR: LazyLock<bool> = LazyLock::new(|| !nocol() && std::io::stderr().is_terminal());

pub fn col(on: bool, c: &str, s: &str) -> String {
	if !on || c.is_empty() { s.into() } else { format!("\x1b[{c}m{s}\x1b[0m") }
}

pub fn link(on: bool, url: &str, s: &str) -> String {
	if on { format!("\x1b]8;;{url}\x1b\\{s}\x1b]8;;\x1b\\") } else { format!("{s} {url}") } // osc 8, plain url when piped
}

pub fn co(c: &str, s: &str) -> String {
	col(*COL_OUT, c, s)
}

pub fn ce(c: &str, s: &str) -> String {
	col(*COL_ERR, c, s)
}

pub fn ok(s: &str) {
	println!("{} {s}", co(GRN, "✓"));
}

pub fn warn(s: &str) {
	let _g = spin::lock(); // clears a running spinner line first, it redraws below
	let cl = if std::io::stderr().is_terminal() { "\r\x1b[K" } else { "" };
	eprintln!("{cl}{} {s}", ce(YLW, "!"));
}

pub fn die(s: &str) -> ! {
	spin::unspin();
	eprintln!("{} {s}", ce(RED, "✗"));
	std::process::exit(1)
}

pub fn n(s: &str) -> usize {
	s.chars().count()
}

pub fn cut(s: &str, w: i64) -> String {
	if w < 1 {
		return String::new(); // to w cols, … if cut
	}
	let w = w as usize;
	if n(s) > w { s.chars().take(w - 1).collect::<String>() + "…" } else { s.into() }
}

pub fn tsize() -> Option<(usize, usize)> {
	crossterm::terminal::size().ok().map(|(w, h)| (w as usize, h as usize)) // cols, rows
}

pub fn bx(t: &str, tc: &str, rows: &[[String; 2]]) {
	let (kw, mut t) = (rows.iter().map(|r| n(&r[0])).max().unwrap_or(0), t.to_string()); // tc = title color, shrinks to the term width
	let mut w = rows.iter().map(|r| kw + 2 + n(&r[1])).max().unwrap_or(0).max(n(&t) + 2);
	if std::io::stdout().is_terminal()
		&& let Some((tw, _)) = tsize()
		&& w + 4 > tw
	{
		w = (tw as i64 - 4).max(kw as i64 + 4) as usize; // wider than the term would wrap every line
		t = cut(&t, w as i64 - 2);
	}
	println!("{}{} {}", co(DIM, "╭─ "), co(&format!("{BOLD};{tc}"), &t), co(DIM, &("─".repeat(w - n(&t) - 1) + "╮")));
	for r in rows {
		let v = cut(&r[1], (w - kw - 2) as i64);
		let vc = if r[0] == t!("k_stat") { tc } else { CYN }; // status row takes the verdict color
		let mut cv = co(vc, &v);
		if r[0] == t!("k_url") && *COL_OUT {
			cv = link(true, &r[1], &cv); // full url stays clickable even when cut
		}
		println!("{}{}  {cv}{}{}", co(DIM, "│ "), co(DIM, &format!("{}{}", r[0], " ".repeat(kw - n(&r[0])))), " ".repeat(w - kw - 2 - n(&v)), co(DIM, " │"));
	}
	println!("{}", co(DIM, &format!("╰{}╯", "─".repeat(w + 2))));
}

#[derive(Clone, Default)]
pub struct Cell {
	pub s: String,
	pub c: String,
}

pub fn cell(s: impl Into<String>, c: &str) -> Cell {
	Cell { s: s.into(), c: c.into() }
}

pub fn tbl(out: &mut String, hd: &[String], rows: &[Vec<Cell>]) {
	let mut w: Vec<usize> = hd.iter().map(|h| n(h)).collect();
	for r in rows {
		for (i, c) in r.iter().enumerate() {
			w[i] = w[i].max(n(&c.s));
		}
	}
	let ln = |out: &mut String, r: &[Cell]| {
		for (i, c) in r.iter().enumerate() {
			let mut s = c.s.clone();
			if i < r.len() - 1 {
				s += &" ".repeat(w[i] - n(&s) + 2);
			}
			*out += &co(&c.c, &s);
		}
		out.push('\n');
	};
	if *COL_OUT && !hd.concat().is_empty() {
		ln(out, &hd.iter().map(|s| cell(s.clone(), &format!("{DIM};{BOLD}"))).collect::<Vec<_>>()); // no header when piped or blank
	}
	for r in rows {
		ln(out, r);
	}
}

static ANSI: LazyLock<Regex> = LazyLock::new(|| Regex::new(r"\x1b\[[0-9;]*m|\x1b\]8;;[^\x1b]*\x1b\\").unwrap());

fn srows(s: &str, w: usize) -> usize {
	s.strip_suffix('\n').unwrap_or(s).split('\n').map(|l| n(&ANSI.replace_all(l, "")).div_ceil(w)).map(|r| r.max(1)).sum() // screen rows incl wrapping
}

pub fn page(hd: &str, s: &str) {
	let mut all = s.to_string(); // hd shown first, then all of it in less if taller than the term
	if !hd.is_empty() {
		println!("{hd}");
		all = format!("{hd}\n{s}");
	}
	match tsize() {
		Some((w, ht)) if std::io::stdout().is_terminal() && w > 0 && srows(&all, w) >= ht => {
			warn(&t!("too_long")); // >= so the prompt still fits
			std::thread::sleep(Duration::from_secs(1));
			if !less(&all) {
				print!("{s}");
			}
		}
		_ => print!("{s}"),
	}
}

pub fn less(s: &str) -> bool {
	if !std::io::stdout().is_terminal() {
		return false;
	}
	let Ok(mut c) = Command::new("less").arg("-R").stdin(Stdio::piped()).spawn() else { return false };
	let _ = c.stdin.take().unwrap().write_all(s.as_bytes());
	c.wait().is_ok_and(|s| s.success())
}

pub fn wrap(s: &str, w: usize) -> Vec<String> {
	let (mut ls, mut l) = (vec![], String::new());
	for x in s.split_whitespace() {
		if !l.is_empty() && n(&l) + 1 + n(x) > w {
			ls.push(std::mem::replace(&mut l, x.into()));
		} else if l.is_empty() {
			l = x.into();
		} else {
			l = l + " " + x;
		}
	}
	ls.push(l);
	ls
}

pub fn quote() {
	let q = if crate::quote::stale() { spin::wait(t!("w_quote"), crate::quote::rand) } else { crate::quote::rand() }; // cached = instant, no spinner blink
	let Some(q) = q else { return };
	let w = tsize().map(|s| s.0).filter(|w| *w >= 20).unwrap_or(80);
	println!();
	for l in wrap(&format!("\"{}\"", q.text), (w - 4).min(76)) {
		println!("  {}", co(&format!("{DIM};3"), &l));
	}
	println!("    {}{}", co(DIM, "- "), co(&format!("{DIM};{CYN}"), &q.author));
}

pub fn score(v: &serde_json::Value) -> String {
	match v {
		serde_json::Value::Number(x) => x.as_f64().map(|f| (f as i64).to_string()).unwrap_or_else(|| x.to_string()), // api gives int, null or raw "int:000100"
		serde_json::Value::String(x) if x.contains(':') => {
			let s = x.split_once(':').unwrap().1.trim_start_matches('0');
			if s.is_empty() { "0".into() } else { s.into() }
		}
		serde_json::Value::String(x) if !x.is_empty() => x.clone(),
		_ => "-".into(),
	}
}

pub fn scol(n: i64) -> &'static str {
	if n < 50 {
		RED // <50 red, <80 yellow, else green
	} else if n < 80 {
		YLW
	} else {
		GRN
	}
}

pub fn scell(s: &str) -> Cell {
	match s.split_whitespace().next().and_then(|x| x.parse::<i64>().ok()) {
		Some(v) => cell(s, &format!("{BOLD};{}", scol(v))), // score cell colored by value, "-" stays dim
		None => cell(s, DIM),
	}
}

pub fn stat(s: &str) -> Cell {
	match s {
		"OK" | "INI_OK" => cell(format!("✓ {s}"), GRN),
		"" | "?" => cell(format!("⋯ {}", t!("pending")), YLW),
		_ => cell(format!("✗ {s}"), RED),
	}
}
