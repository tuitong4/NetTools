#!/bin/sh
. /etc/rc.d/init.d/functions

PATH=/sbin:/usr/sbin:/bin:/usr/bin
DESC="Ping Scheduler"
NAME=nwping-scheduler
DAEMON='/usr/bin/nwping -s -c /etc/nwping/ping_scheduler.conf'
PIDFILE=/var/run/nwping/$NAME.pid
SCRIPTNAME=/etc/init.d/$NAME
USER=root
LOCKFILE=/var/lock/nwping/$NAME.lock


start() {
    echo -n $"Starting $NAME: "
    daemon --user $USER --pidfile $PIDFILE "$DAEMON &>/dev/null & echo \$! > $PIDFILE"
    retval=$?
    echo
    [ $retval -eq 0 ] && touch $LOCKFILE
    return $retval
}

stop() {
    echo -n $"Stopping $NAME: "
    killproc -p $PIDFILE $NAME
    retval=$?
    echo
    [ $retval -eq 0 ] && rm -f $LOCKFILE
    return $retval
}

restart() {
    stop
    start
}

reload() {
    restart
}

force_reload() {
    restart
}

rh_status() {
    status -p $PIDFILE $NAME
}

rh_status_q() {
    rh_status >/dev/null 2>&1
}


case "$1" in
    start)
        rh_status_q && exit 0
        $1
        ;;
    stop)
        rh_status_q || exit 0
        $1
        ;;
    restart)
        $1
        ;;
    reload)
        rh_status_q || exit 7
        $1
        ;;
    force-reload)
        force_reload
        ;;
    status)
        rh_status
        ;;
    condrestart|try-restart)
        rh_status_q || exit 0
        restart
        ;;
    *)
        echo $"Usage: $0 {start|stop|status|restart|condrestart|try-restart|reload|force-reload}"
        exit 2
esac
exit $?

