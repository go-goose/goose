#!/usr/bin/env python

import os
import subprocess
import sys


KNOWN_LIVE_SUITES = [
    'client',
    'glance',
    'identity',
    'nova',
    'neutron',
    'swift',
]


def ensure_juju_core_dependencies():
    """Ensure that juju-core and all dependencies have been installed."""
    # Note: This potentially overwrites goose while it is updating the world.
    # However, if we are targetting the trunk branch of goose, that should have
    # already been updated to the latest version by tarmac.
    # I don't quite see a way to reconcile that we want the latest juju-core
    # and all of the other dependencies, but we don't want to touch goose
    # itself. One option would be to have a split GOPATH. One installs the
    # latest juju-core and everything else. The other is where the
    # goose-under-test resides. So we don't add the goose-under-test to GOPATH,
    # call "go get", then add it to the GOPATH for the rest of the testing.
    cmd = ['go', 'get', '-u', '-x', 'github.com/juju/...']
    sys.stderr.write('Running: %s\n' % (' '.join(cmd),))
    retcode = subprocess.call(cmd)
    if retcode != 0:
        sys.stderr.write('WARN: Failed to update github.com/juju\n')


def setup_gopath():
    pwd = os.getcwd()
    if sys.platform == 'win32':
        pwd = pwd.replace('\\', '/')
    offset = pwd.rfind('src/gopkg.in/goose.v3')
    if offset == -1:
        sys.stderr.write('Could not find "src/gopkg.in/goose.v3" in cwd: %s\n'
                         % (pwd,))
        sys.stderr.write('Unable to automatically set GOPATH\n')
        return
    add_gopath = pwd[:offset].rstrip('/')
    gopath = os.environ.get("GOPATH")
    if gopath:
        if add_gopath in gopath:
            return
        # Put this path first, so we know we are running these tests
        gopath = add_gopath + os.pathsep + gopath
    else:
        gopath = add_gopath
    sys.stderr.write('Setting GOPATH to: %s\n' % (gopath,))
    os.environ['GOPATH'] = gopath


def run_cmd(cmd):
    cmd_str = ' '.join(cmd)
    sys.stderr.write('Running: %s\n' % (cmd_str,))
    retcode = subprocess.call(cmd)
    if retcode != 0:
        sys.stderr.write('FAIL: failed running: %s\n' % (cmd_str,))
    return retcode


def run_live_tests(opts):
    """Run all of the live tests."""
    orig_wd = os.getcwd()
    final_retcode = 0
    for d in KNOWN_LIVE_SUITES:
        try:
            cmd = ['go', 'test', '-v', '-live', '-check.v']
            sys.stderr.write('Running: %s in %s\n' % (' '.join(cmd), d))
            os.chdir(d)
            retcode = subprocess.call(cmd)
            if retcode != 0:
                sys.stderr.write('FAIL: Running live tests in %s\n' % (d,))
                final_retcode = retcode
        finally:
            os.chdir(orig_wd)
    return final_retcode


def main(args):
    import argparse
    p = argparse.ArgumentParser(description='Run the goose test suite')
    p.add_argument('--verbose', action='store_true', help='Be chatty')
    p.add_argument('--version', action='version', version='%(prog)s 0.1')
    p.add_argument('--juju-core', action='store_true',
        help="Run the juju-core trunk tests as well as the goose tests.")
    p.add_argument('--live', action='store_true',
        help="Run tests against a live service.")

    opts = p.parse_args(args)
    setup_gopath()
    to_run = []
    if opts.live:
        to_run.append(run_live_tests)
    for func in to_run:
        retcode = func(opts)
        if retcode != 0:
            return retcode


if __name__ == '__main__':
    import sys
    sys.exit(main(sys.argv[1:]))

