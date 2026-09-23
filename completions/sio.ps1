Register-ArgumentCompleter -Native -CommandName sio -ScriptBlock {
	param($word, $ast, $pos)
	$w = @($ast.CommandElements | % { "$_" })
	$n = $w.Count; if ($word -eq '') { $n++ } # cursor on a new arg
	$c = @()
	if ($n -eq 2) { $c = 'add-host', 'rm-host', 'hosts', 'main', 'token', 'ping', 'info', 'upload', 'probs', 'subs' }
	elseif ($n -eq 3 -and $w[1] -eq 'add-host') { $c = '--main' }
	elseif ($n -eq 3 -and $w[1] -in 'rm-host', 'main', 'token', 'ping', 'info') { $c = sio hosts 2>$null | % { $_.Substring(2).Split(' ')[0] } }
	elseif ($n -ge 4 -and $w[1] -in 'upload', 'up') { $c = Get-ChildItem -Name "$word*" }
	$c | ? { $_ -like "$word*" } | % { [System.Management.Automation.CompletionResult]::new($_) }
}
