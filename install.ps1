# irm https://raw.githubusercontent.com/sioworker/sio-cli/refs/heads/main/install.ps1 | iex
& {
	$ErrorActionPreference = 'Stop'
	$ProgressPreference = 'SilentlyContinue' # iwr's own bar is super slow on ps 5
	[Console]::OutputEncoding = [Text.Encoding]::UTF8
	[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
	$R = 'sioworker/sio-cli'
	$Bin = if ($env:SIO_BIN) { $env:SIO_BIN.TrimEnd('\', '/') } else { Join-Path $env:LOCALAPPDATA 'Programs\sio' }
	$tty = -not [Console]::IsOutputRedirected

	function ok($s) { Write-Host '✓ ' -ForegroundColor Green -NoNewline; Write-Host $s }
	function warn($s) { Write-Host '! ' -ForegroundColor Yellow -NoNewline; Write-Host $s }
	function die($s) { Write-Host '✗ ' -ForegroundColor Red -NoNewline; Write-Host $s; throw 'sio-install' } # no exit, under iex it kills the users shell

	function bar($url, $out) { # download w/ progress bar
		if (-not $tty) { Invoke-WebRequest -UseBasicParsing $url -OutFile $out; return }
		Add-Type -AssemblyName System.Net.Http
		$res = (New-Object System.Net.Http.HttpClient).GetAsync($url, [System.Net.Http.HttpCompletionOption]::ResponseHeadersRead).Result
		if (-not $res.IsSuccessStatusCode) { die "download failed: $url ($([int]$res.StatusCode))" }
		$tot, $got, $last = $res.Content.Headers.ContentLength, 0, 0
		$in, $fs, $buf = $res.Content.ReadAsStreamAsync().Result, [IO.File]::Create($out), (New-Object byte[] 65536)
		try {
			while (($n = $in.Read($buf, 0, $buf.Length)) -gt 0) {
				$fs.Write($buf, 0, $n); $got += $n
				if ($tot -and ([Environment]::TickCount - $last -gt 100 -or $got -eq $tot)) {
					$last, $f = [Environment]::TickCount, [int](30 * $got / $tot)
					Write-Host "`r  " -NoNewline
					Write-Host (('█' * $f) + ('░' * (30 - $f))) -ForegroundColor Cyan -NoNewline
					Write-Host (' {0,3}% {1:N1}/{2:N1} MB' -f [int](100 * $got / $tot), ($got / 1MB), ($tot / 1MB)) -ForegroundColor DarkGray -NoNewline
				}
			}
		} finally { $fs.Close(); $in.Close() }
		Write-Host ''
	}

	try {
		if ($PSVersionTable.PSVersion.Major -ge 6 -and -not $IsWindows) { die 'this is the windows installer, use install.sh: curl -fsSL https://raw.githubusercontent.com/sioworker/sio-cli/refs/heads/main/install.sh | sh' }
		switch ($env:PROCESSOR_ARCHITECTURE) {
			'AMD64' {}
			'ARM64' { warn 'no arm64 build yet, using amd64 (runs under emulation)' }
			default { die "unsupported arch $env:PROCESSOR_ARCHITECTURE" }
		}
		$t = Join-Path ([IO.Path]::GetTempPath()) ("sio-" + [Guid]::NewGuid())
		New-Item -ItemType Directory $t | Out-Null
		try {
			$atom = (Invoke-WebRequest -UseBasicParsing "https://github.com/$R/releases.atom").Content # newest first, no api rate limit, autobuilds are prereleases so no /latest
			if ($atom -notmatch '/releases/tag/([^"]+)"') { die 'no release found' }
			$tag, $f = $Matches[1], 'sio-windows-amd64.exe'
			$tf, $ts, $exe, $cps = (Join-Path $t $f), (Join-Path $t 'sums'), (Join-Path $Bin 'sio.exe'), (Join-Path $Bin 'sio.ps1')
			Write-Host '↓ ' -ForegroundColor Cyan -NoNewline; Write-Host "$f " -NoNewline; Write-Host $tag -ForegroundColor DarkGray
			bar "https://github.com/$R/releases/download/$tag/$f" $tf
			Invoke-WebRequest -UseBasicParsing "https://github.com/$R/releases/download/$tag/sha256sums.txt" -OutFile $ts # to a file, ps 5 gives release assets back as bytes
			$h = (Get-FileHash $tf -Algorithm SHA256).Hash.ToLower()
			if ((Get-Content -Raw $ts) -notmatch "(?m)^$h  $([regex]::Escape($f))\s*$") { die "checksum mismatch for $f" }
			New-Item -ItemType Directory -Force $Bin | Out-Null
			try { Copy-Item -Force $tf $exe } catch { die "cant write $exe, is sio still running? ($_)" }
			ok "installed sio $tag to $exe"

			$up = [Environment]::GetEnvironmentVariable('Path', 'User')
			if (-not $up) { $up = '' } # new accounts can have no user Path at all
			if ((($up -split ';') | ForEach-Object { $_.TrimEnd('\') }) -notcontains $Bin) {
				[Environment]::SetEnvironmentVariable('Path', (($up.TrimEnd(';') + ';' + $Bin).TrimStart(';')), 'User')
				$env:Path += ";$Bin"
				ok "added $Bin to your PATH (new terminals pick it up)"
			}

			Invoke-WebRequest -UseBasicParsing "https://raw.githubusercontent.com/$R/$tag/completions/sio.ps1" -OutFile $cps
			$prof, $line = $PROFILE.CurrentUserAllHosts, ". `"$cps`" # sio completions"
			if ($env:SIO_COMP -eq 'none') {
				warn 'no completions installed'
			} elseif ((Test-Path $prof) -and (Select-String -Quiet -SimpleMatch $cps $prof)) {
				ok "powershell completions (already in $prof)"
			} else {
				$yes = $true
				if (-not $env:SIO_COMP -and $tty -and [Environment]::UserInteractive) { # SIO_COMP=pwsh skips the question
					Write-Host '? ' -ForegroundColor Cyan -NoNewline
					$yes = (Read-Host "add powershell completions to $prof [Y/n]") -notmatch '^[nN]'
				}
				if ($yes) {
					New-Item -ItemType Directory -Force (Split-Path $prof) | Out-Null
					Add-Content $prof $line
					ok "powershell completions (in $prof)"
					if ((Get-ExecutionPolicy -Scope CurrentUser) -eq 'Restricted' -and (Get-ExecutionPolicy) -eq 'Restricted') { warn 'your execution policy blocks profile scripts, run: Set-ExecutionPolicy -Scope CurrentUser RemoteSigned' }
				} else { warn 'no completions installed' }
			}
			ok 'next:'
			ok '	- sio config lang <lang>'
			ok '	- sio config hosts add <name> <domain>'
		} finally { Remove-Item -Recurse -Force $t -ErrorAction SilentlyContinue }
	} catch { if ("$_" -ne 'sio-install') { Write-Host '✗ ' -ForegroundColor Red -NoNewline; Write-Host "$_" } }
}
