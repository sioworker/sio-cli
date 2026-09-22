function __sio_hosts
	sio hosts | string replace -rf '^. (\S+).*' '$1'
end

complete -c sio -f
complete -c sio -n __fish_use_subcommand -a add-host -d 'add a host'
complete -c sio -n __fish_use_subcommand -a rm-host -d 'remove a host'
complete -c sio -n __fish_use_subcommand -a hosts -d 'list hosts'
complete -c sio -n __fish_use_subcommand -a main -d 'set main host'
complete -c sio -n __fish_use_subcommand -a token -d 'set host token'
complete -c sio -n __fish_use_subcommand -a ping -d 'check auth'
complete -c sio -n __fish_use_subcommand -a upload -d 'submit a solution'
complete -c sio -n '__fish_seen_subcommand_from add-host' -l main -s m -d 'make it the main host'
complete -c sio -n '__fish_seen_subcommand_from rm-host main token ping' -a '(__sio_hosts)' -d host
complete -c sio -n '__fish_seen_subcommand_from upload up; and test (count (commandline -opc)) -ge 3' -F
