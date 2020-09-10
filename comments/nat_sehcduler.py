from __future__ import print_function
import subprocess

BIRD_PATH      = ""
BIRD_CONF_PATH = ""
BIRD_CTL_SOCK_FILE = ""
BIRD_PROC = "bird"
BIRD_CTL_PROC = "birdc"
BIRD_CONF_FILE = \
    {
        "BJ03-to-BJ04" : "bj03_to_bj04.conf",
        "BJ04-to-BJ05" : "bj04_to_bj03.conf"
    }

def exec_sys_cmd(cmd):
    cmds = cmd.strip().split()
    return subprocess.check_output(cmds)

def is_proc_running(proc):
    o = exec_sys_cmd("pidof %s" % proc)
    o = o.strip()
    if  o != "":
        return True, o 
    else:
        return False, None

def kill_proc(pid):
    return exec_sys_cmd("kill -9 %s" % pid)

def stop_bgp_proc():
    run, pid = is_proc_running(BIRD_PROC)
    if run:
        print("%s is running, going to stop." % BIRD_PROC)
        kill_proc(pid)
        print("%s is stopped." % BIRD_PROC)

def start_bgp_proc(op_type):
    if op_type not in BIRD_CONF_FILE:
        raise ValueError("Operation '%s' is not supported now." % op_type)
    
    cmd = BIRD_PATH + "/" + BIRD_PROC
    cmd += " -c " + BIRD_CONF_PATH + "/" + BIRD_CONF_FILE[op_type]
    
    if BIRD_CTL_SOCK_FILE:
        cmd += " -s " + BIRD_CTL_SOCK_FILE

    o = exec_sys_cmd(cmd)
    print("%s is started: %s" % (BIRD_PROC, o))

def send_route(bgp_instance, route, preference, next_hop, community):
    _route      = '\"%s\"' % route
    _preference = '\"%s\"' % preference
    _next_hop   = '\"%s\"' % next_hop
    _community  = '\"%s\"' % community

    cmd = BIRD_PATH + "/" + BIRD_CTL_PROC
    cmd += " rswitch " + bgp_instance
    cmd += " "

    o = exec_sys_cmd(cmd)
    print("Route '%s' is advertismented with preference '%s', community '%s': %s" % \
                (route, preference, community, o))

def withdraw_route(bgp_instance, route, preference, next_hop, community):
    _route      = '\"%s\"' % route
    _preference = '\"%s\"' % preference
    _next_hop   = '\"%s\"' % next_hop
    _community  = '\"%s\"' % community

    cmd = BIRD_PATH + "/" + BIRD_CTL_PROC
    cmd += " rswitch " + bgp_instance
    cmd += " "

    o = exec_sys_cmd(cmd)
    print("Route '%s' is withdrawed: %s" % (route, o)

