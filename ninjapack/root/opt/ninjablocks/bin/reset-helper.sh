#!/bin/sh

die() {
	echo "$*" 1>&2
	exit 1
}

factory_reset() {
	service sphere-client stop
	service sphere-director stop
	service ledcontroller stop
	"$(dirname "$0")/recovery.sh" with media-updated choose-latest factory-reset "$@"
}

main()
{
	cmd=$1
	case "$1" in
	init-io)
		if test -e /sys/kernel/debug/omap_mux/xdma_event_intr1; then
			# in later kernels (3.12) this path won't exist
			echo 37 > /sys/kernel/debug/omap_mux/xdma_event_intr1
		fi

		rc=0

		echo "reset-helper.sh: programming gpio20 begins..."
		echo 20 > /sys/class/gpio/export || rc=$? || echo "warning: 'echo > /sys/class/gpio/export failed with $rc." 1>&2
		echo in > /sys/class/gpio/gpio20/direction || rc=$? || echo "warning: 'echo > /sys/class/gpio/gpio20/direction' failed with $rc." 1>&2
		echo "reset-helper.sh: programming gpio20 ends..."
		exit $rc

		# to read the reset button
		# cat /sys/class/gpio/gpio20/value
	;;
	reboot)
		sync
		/sbin/reboot
	;;
	reset-userdata)
		# TBD: write scripts that will reset the user-data
		sphere-reset --reset-setup
		sync
		/sbin/reboot
	;;
	reset-root)
		shift 1
		# in this phase, we just exit with 0
		# the setup assistant binary will exit with 168 which will cause
		# it's wrapper script to start a factory reset
	;;
	factory-reset)
		shift 1
		factory_reset "$@"
	;;
	esac
}

main "$@"
