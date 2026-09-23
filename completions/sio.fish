function __sio_hosts
	sio config hosts | string replace -rf '^. (\S+).*' '$1'
end

function __sio_langs
	sio config lang | string replace -rf '^. (\S+).*' '$1'
end

function __sio_at # typed words after sio == argv
	set -l w (commandline -opc)
	test "$w[2..]" = "$argv"
end

complete -c sio -f
complete -c sio -n __fish_use_subcommand -a config -d 'hosts and language'
complete -c sio -n __fish_use_subcommand -a ping -d 'check auth'
complete -c sio -n __fish_use_subcommand -a info -d 'show contests'
complete -c sio -n __fish_use_subcommand -a tree -d 'contests and their problems'
complete -c sio -n __fish_use_subcommand -a upload -d 'submit a solution'
complete -c sio -n __fish_use_subcommand -a probs -d 'list problems'
complete -c sio -n __fish_use_subcommand -a subs -d 'list own submissions'
complete -c sio -n '__sio_at config' -a hosts -d 'list/add/remove hosts'
complete -c sio -n '__sio_at config' -a lang -d 'set language'
complete -c sio -n '__sio_at config' -a quotes -d 'random quote after cmds'
complete -c sio -n '__sio_at config quotes' -a 'on off'
complete -c sio -n '__sio_at config hosts' -a add -d 'add a host'
complete -c sio -n '__sio_at config hosts' -a rm -d 'remove a host'
complete -c sio -n '__sio_at config hosts' -a main -d 'set main host'
complete -c sio -n '__sio_at config hosts' -a token -d 'set host token'
complete -c sio -n '__sio_at config hosts add' -l main -s m -d 'make it the main host'
complete -c sio -n '__sio_at config hosts rm; or __sio_at config hosts main; or __sio_at config hosts token; or __sio_at ping; or __sio_at info; or __sio_at tree' -a '(__sio_hosts)' -d host
complete -c sio -n '__sio_at config lang' -a '(__sio_langs) auto' -d lang
complete -c sio -n '__fish_seen_subcommand_from upload up; and test (count (commandline -opc)) -ge 3' -F
