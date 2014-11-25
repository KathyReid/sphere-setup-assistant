#!/bin/sh

die() {
	echo "$*" 1>&2
	exit 1
}

# setup the recovery environment. look for an environment far on the image partition and use it, if it exists.
setup() {
	RECOVERY_IMAGE_DEVICE=${RECOVERY_IMAGE_DEVICE:-/dev/mmcblk0p4}

	if mountpoint=$(mount_helper require-mounted "${RECOVERY_IMAGE_DEVICE}"); then
		if test -f "$mountpoint/recovery.env.sh"; then
			echo "info: found '$mountpoint/recovery.env.sh' - loading..." 1>&2
			. "$mountpoint/recovery.env.sh"
		else
			echo "info: no overrides found in '$mountpoint/recovery.env.sh' - using defaults" 1>&2
		fi
	else
		echo "warning: could not find recovery image device - using defaults" 1>&2
	fi
}

#
# provides two functions to support mounting of a device
#
mount_helper() {

	cmd=$1
	device=$2
	mountpoint=${3:-/tmp/image}
	case "$cmd" in
	mount-point)
		df | tr -s ' ' | cut -f1,6 -d' ' | grep "^$device" | cut -f2 -d' '
	;;
	require-mounted)
		current=$(mount_helper mount-point "$device")

		if test -z "$current"; then
			test -d "$mountpoint" || mkdir -p "$mountpoint" &&
			/bin/mount "$device" "$mountpoint" &&
			current=$(mount_helper mount-point "$device")
		fi

		if test -n "$current"; then
			echo "$current"
			return 0
		else
			return 1
		fi
	;;
	esac
}

#
# provide functions that answer parts of a URL
#
url() {
	id=$1
	case "$id" in
	prefix)
		echo ${RECOVERY_PREFIX:-https://firmware.sphere.ninja/latest}
	;;
	image)
		echo ${RECOVERY_IMAGE:-ubuntu_armhf_trusty_release_sphere-stable}
	;;
	suffix)
		echo ${RECOVERY_SUFFIX:--recovery}$2
	;;
	url)
		echo $(url prefix)/$(url image)$(url suffix "$2")
	;;
	esac
}

# OSX basename doesn't like -recovery in the basename unless -s is used, but Linux is ok with it
gnu_basename()
{
	case "$(uname)" in
	"Darwin")
		basename -s "$2" "$1"
	;;
	*)
		basename "$1" "$2"
	;;
	esac
}

# NAND image doesn't have sha1sum, but does have openssl sha1, which exists elsewhere too
sha1() {
	openssl sha1 | sed "s/.*= //"
}

# check that contents of a file has the same sha1sum as the contents of a co-located .sha1 file
check_file() {
	file=$1
	filesum="$(sha1 < "${file}")"
	checksum="$(cat "${file}.sha1")"
	test "$filesum" = "$checksum" || die "checksum failed: '$file' $filesum != $checksum"
}

# download the recovery script and report the location of the downloaded file
download_recovery_script() {
	sha1name=/tmp/$(url image)$(url suffix .sh.sha1)
	shname=/tmp/$(url image)$(url suffix .sh)

	! test -f "$sha1name" || rm "$sha1name" || die "could not delete existing sha1 file - $sha1name"
	! test -f "$shname" || rm "$shname" || die "could not delete existing sh file - $shname"

	curl -s "$(url url .sh.sha1)" > "$sha1name" &&
	curl -s "$(url url .sh)" > "$shname" &&
	check_file "$shname" &&
	echo $shname || die "failed to download '$(url url .sh)' to '$shname'"
}

# checks that we are in at least 2014
check_time() {
	year=$(date +%Y)
	test "$year" -ge 2014 || die "bad clock state: $(date "+%Y-%m-%d %H:%M:%S")"
}

# if the specified recovery image exists in the image mountpoint use it. if that image doesn't
# exist, look for a -recovery.tar and use that instead.
image_from_mount_point() {
	local mountpoint=$1
	if test -f "$mountpoint/${RECOVERY_IMAGE}/$(url suffix .tar)"; then
		echo "${RECOVERY_IMAGE}"
	else
		basename=$(gnu_basename "$(ls -d $mountpoint/*-recovery.tar | sort | tail -1)" "$(url suffix .tar)")
		if test -n "$basename"; then
			echo "$basename"
		else
			echo "${RECOVERY_IMAGE}"
		fi
	fi
}

# initiate the factory reset
factory_reset() {
	# check_time

	if recovery_script=$(download_recovery_script) && test -f "$recovery_script"; then
		sh "$recovery_script" recovery-with-network
	else
		if ! mountpoint="$(mount_helper require-mounted "${RECOVERY_IMAGE_DEVICE}")"; then
			die "unable to mount recovery image device: ${RECOVERY_IMAGE_DEVICE}"
		else
			RECOVERY_IMAGE=$(image_from_mount_point "$mountpoint")
			script_file="$(url image)$(url suffix .sh)"
			sha1_file="$(url image)$(url suffix .sh.sha1)"
			tar="$mountpoint/$(url image)$(url suffix .tar)"
			unpacked_script="/tmp/${script_file}"
			unpacked_sha1="/tmp/${sha1_file}"
			if test -n "$tar"; then
				tar -O -xf "$tar" "${script_file}" > "${unpacked_script}" &&
				tar -O -xf "$tar" "${sha1_file}" > "${unpacked_sha1}" &&
				check_file "${unpacked_script}" &&
				sh "${unpacked_script}" recovery-without-network "$tar"
			else
				die "could not locate recovery tar on recovery image device"
			fi
		fi
	fi
}

main()
{
	setup
	cmd=$1
	case "$1" in
	init-io)
		if test -e /sys/kernel/debug/omap_mux/xdma_event_intr1; then
			# in later kernels (3.12) this path won't exist
			echo 37 > /sys/kernel/debug/omap_mux/xdma_event_intr1
			echo 20 > /sys/class/gpio/export
			echo in > /sys/class/gpio/gpio20/direction
		fi

		# to read the reset button
		# cat /sys/class/gpio/gpio20/value
	;;
	reboot)
		/sbin/reboot
	;;
	reset-userdata)
		# TBD: write scripts that will reset the user-data
		sphere-reset --reset-setup
		/sbin/reboot
	;;
	reset-root)
		# TBD: write scripts that will reset the root partition
		sphere-reset --reset-setup
		/sbin/reboot
	;;
	factory-reset)
		shift 1
		factory_reset "$@"
	;;
	download-recovery-script)
		shift 1
		download_recovery_script "$@"
	;;
	image-from-mount-point)
		shift 1
		image_from_mount_point "$@"
	;;
	url)
		url "$@"
	;;
	esac

}


main "$@"
