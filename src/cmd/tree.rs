use crate::cfg::Cfg;
use crate::cmd::probs::{fetch, tex};
use crate::out::{BOLD, COL_OUT, CYN, DIM, RED, ce, co, die, link, warn};
use crate::{fail, host, spin, t};
use std::{
	io::IsTerminal,
	sync::atomic::{AtomicUsize, Ordering},
};

pub fn run(c: &Cfg, args: &[String]) {
	let (n, h) = host(c, args.first().map_or("", |s| s));
	let na = || -> ! { die(&t!("tree_na", ce(CYN, &n))) };
	let cs = match spin::wait(t!("w_contests", n), || h.contests()) {
		Err(e) if e.code() == 404 => na(),
		Err(e) => fail(&n, &e),
		Ok(cs) => cs,
	};
	if cs.is_empty() {
		warn(&t!("no_contests", ce(CYN, &n)));
		return;
	}
	let first = match spin::wait(t!("w_probs", cs[0].id), || h.probs(&cs[0].id)) {
		Err(e) if e.code() == 404 => na(), // no problem_list on old oioioi
		r => r.ok(),
	};
	if std::io::stdin().is_terminal() && std::io::stdout().is_terminal() {
		crate::tui::run(&n, &h, cs, first);
		return;
	}
	let dn = AtomicUsize::new(0);
	let rs: Vec<_> = spin::wait(t!("w_probs_n", format!("0/{}", cs.len())), || {
		std::thread::scope(|s| {
			let hs: Vec<_> = cs
				.iter()
				.map(|ct| {
					let (h, dn, l) = (&h, &dn, cs.len());
					s.spawn(move || {
						let r = fetch(h, &ct.id).0;
						spin::set(t!("w_probs_n", format!("{}/{l}", dn.fetch_add(1, Ordering::Relaxed) + 1)));
						r
					})
				})
				.collect();
			hs.into_iter().map(|j| j.join().unwrap()).collect()
		})
	});
	let lk = |u: &str, s: &str| if *COL_OUT { link(true, u, s) } else { s.into() }; // clickable in a tty, plain when piped
	let mut b = String::new();
	for (i, (ct, r)) in cs.iter().zip(rs).enumerate() {
		if i > 0 {
			b.push('\n');
		}
		b += &format!("{} {}\n", co(&format!("{BOLD};{CYN}"), &lk(&format!("{}/c/{}/", h.url, ct.id), &ct.id)), co(DIM, &ct.name));
		let ps = match r {
			Err(e) => {
				b += &format!("{}{}\n", co(DIM, "└── "), co(RED, &format!("✗ {e}")));
				continue;
			}
			Ok(ps) => ps,
		};
		if ps.is_empty() {
			b += &format!("{}\n", co(DIM, &format!("└── {}", t!("empty"))));
		}
		let w = ps.iter().map(|p| p.short.len()).max().unwrap_or(0);
		for (j, p) in ps.iter().enumerate() {
			let br = if j == ps.len() - 1 { "└── " } else { "├── " };
			b += &format!("{}{}\n", co(DIM, br), lk(&format!("{}/c/{}/p/{}/", h.url, ct.id, p.short), &format!("{}  {}", co(CYN, &format!("{:w$}", p.short)), tex(&p.name))));
		}
	}
	print!("{b}");
}
