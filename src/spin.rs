use crate::out::{CYN, DIM, ce, cut, tsize};
use std::{
	io::IsTerminal,
	sync::{
		Arc, Mutex, MutexGuard,
		atomic::{AtomicBool, Ordering},
	},
	thread::{self, JoinHandle},
	time::Duration,
};

static ERR: Mutex<()> = Mutex::new(()); // spinner + warn/die share stderr
static MSG: Mutex<String> = Mutex::new(String::new());
static RUN: Mutex<Option<(Arc<AtomicBool>, JoinHandle<()>)>> = Mutex::new(None);

fn lk<T>(m: &Mutex<T>) -> MutexGuard<'_, T> {
	m.lock().unwrap_or_else(|e| e.into_inner())
}

pub fn lock() -> MutexGuard<'static, ()> {
	lk(&ERR)
}

pub fn wait<T>(msg: String, f: impl FnOnce() -> T) -> T {
	if !std::io::stderr().is_terminal() {
		return f(); // spinner on stderr while f runs, tty only
	}
	*lk(&MSG) = msg;
	let s2 = Arc::new(AtomicBool::new(false));
	let st = s2.clone();
	let h = thread::spawn(move || {
		let (fr, mut i) = (["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"], 0);
		loop {
			{
				let _g = lock();
				let mut m = lk(&MSG).clone();
				if let Some((w, _)) = tsize() {
					m = cut(&m, w as i64 - 3); // one row only, \x1b[K cant clear a wrapped line
				}
				eprint!("\r\x1b[K{} {}", ce(CYN, fr[i % fr.len()]), ce(DIM, &m));
			}
			i += 1;
			for _ in 0..8 {
				thread::sleep(Duration::from_millis(10));
				if st.load(Ordering::Relaxed) {
					let _g = lock();
					eprint!("\r\x1b[K");
					return;
				}
			}
		}
	});
	*lk(&RUN) = Some((s2, h));
	let r = f();
	unspin();
	r
}

pub fn unspin() {
	if let Some((s, h)) = lk(&RUN).take() {
		s.store(true, Ordering::Relaxed);
		let _ = h.join();
	}
}

pub fn set(s: String) {
	*lk(&MSG) = s;
}
