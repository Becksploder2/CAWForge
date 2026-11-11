"""CAWForge isolated import diagnostic: detect broken DLL imports and hung imports.

Usage: python check_render_imports.py --timeout 12 --json
Does not install or modify dependencies.
"""
import argparse
import json
import subprocess
import sys
import time

MODULES = ('numpy', 'PIL', 'torch', 'onnxruntime', 'rembg')


def probe(module, timeout=12, runner=subprocess.run, executable=None):
    if not module.isidentifier():
        raise ValueError('Invalid module name')
    command = [executable or sys.executable, '-c',
               'import importlib; importlib.import_module(%r); print("IMPORT_OK")' % module]
    start = time.monotonic()
    try:
        result = runner(command, capture_output=True, text=True, timeout=timeout)
    except subprocess.TimeoutExpired:
        return {'module': module, 'status': 'timeout', 'seconds': round(time.monotonic()-start, 2)}
    except OSError as exc:
        return {'module': module, 'status': 'launch_error', 'error': str(exc)}
    status = 'ok' if result.returncode == 0 and 'IMPORT_OK' in result.stdout else 'import_error'
    return {'module': module, 'status': status, 'seconds': round(time.monotonic()-start, 2),
            'details': (result.stderr or result.stdout)[-1500:] if status != 'ok' else ''}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--timeout', type=float, default=12)
    parser.add_argument('--json', action='store_true')
    parser.add_argument('--modules', nargs='+', default=list(MODULES), choices=MODULES)
    args = parser.parse_args(argv)
    if args.timeout <= 0 or args.timeout > 120:
        parser.error('--timeout must be greater than zero and at most 120 seconds')
    results = [probe(module, args.timeout) for module in args.modules]
    report = {'python': sys.executable, 'results': results}
    if args.json:
        print(json.dumps(report, indent=2))
    else:
        print('CAWForge isolated import diagnostic using', sys.executable)
        for item in results:
            print(item['module'], item['status'], item.get('seconds', '?'), 'seconds')
            if item.get('details'):
                print(item['details'])
        print('A successful import does not prove the GPU renderer works.')
    return 0 if all(r['status'] == 'ok' for r in results) else 1


if __name__ == '__main__':
    raise SystemExit(main())
