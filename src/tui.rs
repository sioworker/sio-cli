use crate::api::{Contest, Prob};
use crate::cfg::Host;
use crate::cmd::probs::{fetch, tex};
use crate::cmd::up::{NO_API, NO_SESS, dur, guess, judge, verdict};
use crate::out::{BOLD, COL_OUT, CYN, DIM, GRN, RED, YLW, co, die, link, n};
use crate::t;
use crossterm::event::{self, Event, KeyCode, KeyEvent, KeyEventKind, KeyModifiers};
use std::{
	io::Write,
	sync::{Arc, Mutex},
	thread,
	time::Duration,
};

#[derive(Clone)]
struct Seg(String, String); // text, color

fn sg(s: impl Into<String>, c: &str) -> Seg {
	Seg(s.into(), c.into())
}

fn fit(ss: &[Seg], mut w: usize) -> String {
	let mut o = String::new(); // cut to w cols, colors kept
	for g in ss {
		if w == 0 {
			break;
		}
		let s: String = g.0.chars().take(w).collect();
		w -= n(&s);
		o += &co(&g.1, &s);
	}
	o
}

fn browse(u: &str) {
	let (c, a): (&str, Vec<&str>) = if cfg!(target_os = "macos") {
		("open", vec![u])
	} else if cfg!(windows) {
		("rundll32", vec!["url.dll,FileProtocolHandler", u])
	} else {
		("xdg-open", vec![u])
	};
	if let Ok(mut ch) = std::process::Command::new(c).args(a).stdout(std::process::Stdio::null()).stderr(std::process::Stdio::null()).spawn() {
		thread::spawn(move || ch.wait());
	}
}

struct Node {
	ct: Contest,
	open: bool,
	busy: bool,
	ps: Option<Vec<Prob>>, // None = not loaded yet
	err: Option<String>,
}

struct Inp {
	c: usize,
	p: usize,
	buf: String,
}

struct St {
	ns: Vec<Node>,
	cur: i64,
	top: i64,
	rev: Option<usize>, // contest to scroll into view once opened
	inp: Option<Inp>,   // file prompt after u
	busy: bool,
	up_msg: String,
	note: Seg,
	dirty: bool,
}

#[derive(Clone, Copy)]
struct Row(usize, i64); // contest, prob (-1 = contest, -2 = status line)

impl St {
	fn rows(&self) -> Vec<Row> {
		let mut rs = vec![];
		for (i, n) in self.ns.iter().enumerate() {
			rs.push(Row(i, -1));
			if !n.open {
				continue;
			}
			let ps = n.ps.as_deref().unwrap_or(&[]);
			if n.busy || n.err.is_some() || ps.is_empty() {
				rs.push(Row(i, -2));
			}
			rs.extend((0..ps.len()).map(|j| Row(i, j as i64)));
		}
		rs
	}
}

struct Guard; // puts the term back even on panic

impl Drop for Guard {
	fn drop(&mut self) {
		print!("\x1b[?25h\x1b[?1049l");
		let _ = std::io::stdout().flush();
		let _ = crossterm::terminal::disable_raw_mode();
	}
}

fn lk(m: &Mutex<St>) -> std::sync::MutexGuard<'_, St> {
	m.lock().unwrap_or_else(|e| e.into_inner())
}

const FR: [&str; 10] = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"];

pub fn run(hn: &str, h: &Host, cs: Vec<Contest>, first: Option<Vec<Prob>>) {
	if let Err(e) = crossterm::terminal::enable_raw_mode() {
		die(&e.to_string());
	}
	let _g = Guard;
	print!("\x1b[?1049h\x1b[?25l"); // alt screen, hide cursor
	let mut ns: Vec<Node> = cs.into_iter().map(|ct| Node { ct, open: false, busy: false, ps: None, err: None }).collect();
	ns[0].ps = first;
	let st = Arc::new(Mutex::new(St { ns, cur: 0, top: 0, rev: None, inp: None, busy: false, up_msg: String::new(), note: sg("", ""), dirty: true }));
	let (hn, h) = (hn.to_string(), Arc::new(h.clone()));
	let mut last = (0usize, 0usize);
	loop {
		let sz = crate::out::tsize().unwrap_or((80, 24));
		{
			let mut s = lk(&st);
			if s.dirty || s.busy || sz != last {
				draw(&mut s, &hn, &h, sz);
				s.dirty = false;
				last = sz;
			}
		}
		if !event::poll(Duration::from_millis(100)).unwrap_or(false) {
			continue;
		}
		let Ok(ev) = event::read() else { return };
		let Event::Key(k) = ev else {
			lk(&st).dirty = true; // resize etc
			continue;
		};
		if k.kind == KeyEventKind::Release {
			continue;
		}
		if !key(&st, &hn, &h, k, sz.1) {
			return;
		}
	}
}

fn draw(s: &mut St, hn: &str, h: &Host, (w, ht): (usize, usize)) {
	let rs = s.rows();
	let bh = ht.saturating_sub(2).max(1) as i64;
	s.cur = s.cur.clamp(0, rs.len() as i64 - 1);
	if s.cur < s.top {
		s.top = s.cur;
	}
	if s.cur >= s.top + bh {
		s.top = s.cur - bh + 1;
	}
	if let Some(rv) = s.rev {
		let mut l = s.cur;
		while ((l + 1) as usize) < rs.len() && rs[(l + 1) as usize].0 == rv {
			l += 1;
		}
		if l >= s.top + bh {
			s.top = s.cur.min(l - bh + 1);
		}
		if !s.ns[rv].busy {
			s.rev = None;
		}
	}
	let mut b = format!("\x1b[H{}\x1b[K\r\n", fit(&[sg("✓ ", GRN), sg(hn, &format!("{BOLD};{CYN}")), sg(format!(" {}", h.url), DIM)], w));
	for i in s.top..s.top + bh {
		if let Some(&Row(c, p)) = rs.get(i as usize) {
			let mk = if i == s.cur { sg("❯ ", &format!("{BOLD};{CYN}")) } else { sg("  ", "") };
			let nd = &s.ns[c];
			if p == -1 {
				let cst = if i == s.cur { format!("{BOLD};{CYN}") } else { CYN.into() };
				b += &fit(&[mk, sg(if nd.open { "▾ " } else { "▸ " }, DIM), sg(&nd.ct.id, &cst), sg(format!(" {}", nd.ct.name), DIM)], w);
			} else if p == -2 {
				let x = if nd.busy {
					sg(format!("⋯ {}", t!("loading")), YLW)
				} else if let Some(e) = &nd.err {
					sg(format!("✗ {e}"), RED)
				} else {
					sg(t!("empty"), DIM)
				};
				b += &fit(&[mk, sg("  └── ", DIM), x], w);
			} else {
				let ps = nd.ps.as_deref().unwrap_or(&[]);
				let (pr, pw) = (&ps[p as usize], ps.iter().map(|q| q.short.len()).max().unwrap_or(0));
				let br = if p as usize == ps.len() - 1 { "  └── " } else { "  ├── " };
				let nm = sg(tex(&pr.name), if i == s.cur { BOLD } else { "" });
				b += &link(*COL_OUT, &format!("{}/c/{}/p/{}/", h.url, nd.ct.id, pr.short), &fit(&[mk, sg(br, DIM), sg(format!("{:pw$}", pr.short), CYN), sg("  ", ""), nm], w));
			}
		}
		b += "\x1b[K\r\n";
	}
	let ft = if let Some(inp) = &s.inp {
		let nd = &s.ns[inp.c];
		let pr = &nd.ps.as_deref().unwrap_or(&[])[inp.p];
		vec![sg(format!("↑ {}", t!("up_ask", format!("{}/{}", nd.ct.id, pr.short))), CYN), sg(&inp.buf, ""), sg("█", DIM)]
	} else if s.busy {
		let fr = FR[(std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH).unwrap_or_default().as_millis() / 100 % 10) as usize];
		let mut v = vec![sg(format!("{fr} "), CYN), sg(&s.up_msg, DIM)];
		if !s.note.0.is_empty() {
			v.push(sg(format!("  {}", s.note.0), &s.note.1));
		}
		v
	} else if !s.note.0.is_empty() {
		vec![s.note.clone()]
	} else {
		vec![sg(t!("tree_help"), DIM)]
	};
	b += &format!("{}\x1b[K\x1b[J", fit(&ft, w));
	let mut o = std::io::stdout().lock();
	let _ = o.write_all(b.as_bytes());
	let _ = o.flush();
}

fn open(st: &Arc<Mutex<St>>, s: &mut St, i: usize, h: &Arc<Host>) {
	let nd = &mut s.ns[i];
	nd.open = true;
	s.rev = Some(i);
	if nd.ps.is_some() || nd.busy || nd.err.is_some() {
		return;
	}
	nd.busy = true;
	let (st, h, ct) = (st.clone(), h.clone(), nd.ct.id.clone());
	thread::spawn(move || {
		let r = fetch(&h, &ct).0;
		let mut s = lk(&st);
		let nd = &mut s.ns[i];
		match r {
			Ok(ps) => nd.ps = Some(ps),
			Err(e) => nd.err = Some(e.to_string()),
		}
		nd.busy = false;
		s.dirty = true;
	});
}

fn send(st: &Arc<Mutex<St>>, s: &mut St, hn: &str, h: &Arc<Host>) {
	let Some(inp) = s.inp.take() else { return }; // upload inp.buf in the bg
	let nd = &s.ns[inp.c];
	let (ct, p, f) = (nd.ct.id.clone(), nd.ps.as_deref().unwrap_or(&[])[inp.p].short.clone(), inp.buf.trim().to_string());
	if f.is_empty() {
		s.note = sg(format!("✗ {}", t!("no_file", p)), RED);
		return;
	}
	(s.busy, s.up_msg) = (true, t!("w_up", f, format!("{ct}/{p}")));
	let (st, h, hn) = (st.clone(), h.clone(), hn.to_string());
	thread::spawn(move || {
		let note = match h.submit(&ct, &p, &f) {
			Ok(id) => {
				lk(&st).up_msg = t!("w_judge", p, "0s"); // wait for the verdict, like sio submit
				let (sb, _) = judge(&h, &ct, &p, &id, |d| lk(&st).up_msg = t!("w_judge", p, dur(d)));
				let (mut v, vc) = verdict(&sb);
				if sb.status == NO_API {
					v += &format!(", {}", t!("need_login", format!("sio config hosts login {hn}")));
				} else if sb.status == NO_SESS {
					v += &format!(", {}", t!("sess_exp", hn, format!("sio config hosts login {hn}")));
				}
				sg(format!("{v}  {}", t!("up_ok", f, format!("{ct}/{p}"), id)), &vc)
			}
			Err(e) => sg(format!("✗ {e}"), RED),
		};
		let mut s = lk(&st);
		(s.busy, s.note, s.dirty) = (false, note, true);
	});
}

fn key(st: &Arc<Mutex<St>>, hn: &str, h: &Arc<Host>, k: KeyEvent, ht: usize) -> bool {
	let mut s = lk(st); // false = quit
	s.dirty = true;
	let ctrl = k.modifiers.contains(KeyModifiers::CONTROL);
	if ctrl && k.code == KeyCode::Char('c') {
		return false;
	}
	if s.inp.is_some() {
		match k.code {
			KeyCode::Esc => s.inp = None, // typing the file path
			KeyCode::Enter => send(st, &mut s, hn, h),
			KeyCode::Backspace => {
				s.inp.as_mut().unwrap().buf.pop();
			}
			KeyCode::Char(c) if !ctrl => s.inp.as_mut().unwrap().buf.push(c),
			_ => {}
		}
		return true;
	}
	let rs = s.rows();
	let bh = ht.saturating_sub(2).max(1) as i64;
	let Row(rc, rp) = rs[s.cur.clamp(0, rs.len() as i64 - 1) as usize];
	if !s.busy {
		s.note = sg("", "");
	}
	match k.code {
		KeyCode::Char('q') | KeyCode::Esc => {
			if s.busy {
				s.note = sg(format!("! {}", t!("up_busy")), YLW); // quitting would kill the upload mid way
			} else {
				return false;
			}
		}
		KeyCode::Char('u') => {
			if s.busy {
				s.note = sg(format!("! {}", t!("up_busy")), YLW);
			} else if rp >= 0 {
				let f = guess(&s.ns[rc].ps.as_deref().unwrap_or(&[])[rp as usize].short).map(|g| g.1).unwrap_or_default();
				s.inp = Some(Inp { c: rc, p: rp as usize, buf: f });
			} else {
				s.note = sg(format!("! {}", t!("up_pick")), YLW);
			}
		}
		KeyCode::Up | KeyCode::Char('k') => s.cur -= 1,
		KeyCode::Down | KeyCode::Char('j') => s.cur += 1,
		KeyCode::PageUp => s.cur -= bh,
		KeyCode::PageDown => s.cur += bh,
		KeyCode::Home | KeyCode::Char('g') => s.cur = 0,
		KeyCode::End | KeyCode::Char('G') => s.cur = rs.len() as i64 - 1,
		KeyCode::Right | KeyCode::Char('l') => {
			if rp == -1 {
				open(st, &mut s, rc, h);
			}
		}
		KeyCode::Left | KeyCode::Char('h') => {
			if rp == -1 {
				s.ns[rc].open = false;
			} else {
				while s.cur > 0 && rs[s.cur as usize].1 != -1 {
					s.cur -= 1;
				}
			}
		}
		KeyCode::Char(' ') | KeyCode::Enter => {
			if rp == -1 {
				if s.ns[rc].open {
					s.ns[rc].open = false;
				} else {
					open(st, &mut s, rc, h);
				}
			} else if rp >= 0 {
				let nd = &s.ns[rc];
				browse(&format!("{}/c/{}/p/{}/", h.url, nd.ct.id, nd.ps.as_deref().unwrap_or(&[])[rp as usize].short));
			}
		}
		_ => {}
	}
	true
}
