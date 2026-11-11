"""Non-destructive CAWForge render environment diagnostic.

Run with the exact Python executable used to launch render_cutout.py:
    py -3 check_render_environment.py --script E:\\CAWForge\\render_cutout.py
"""
import argparse
import importlib.util
import json
from pathlib import Path
import sys

REQUIRED = ('torch', 'numpy', 'onnxruntime', 'rembg')


def inspect_environment(script=None, finder=None, executable=None):
    finder = finder or importlib.util.find_spec
    executable = executable or sys.executable
    modules = {}
    for name in REQUIRED:
        try:
            spec = finder(name)
            modules[name] = {'available': spec is not None}
        except (ImportError, ValueError, ModuleNotFoundError) as exc:
            modules[name] = {'available': False, 'error': str(exc)}
    script_path = Path(script).expanduser() if script else None
    return {
        'python_executable': executable,
        'python_version': sys.version.split()[0],
        'script_path': str(script_path) if script_path else None,
        'script_exists': script_path.is_file() if script_path else None,
        'modules': modules,
        'all_modules_found': all(v['available'] for v in modules.values()),
    }


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--script', help='Absolute path to render_cutout.py')
    parser.add_argument('--json', action='store_true', help='Print machine-readable report')
    args = parser.parse_args(argv)
    result = inspect_environment(args.script)
    if args.json:
        print(json.dumps(result, indent=2))
    else:
        print('Python:', result['python_executable'], '(' + result['python_version'] + ')')
        if args.script:
            print('Render script:', result['script_path'], 'FOUND' if result['script_exists'] else 'MISSING')
        for name, status in result['modules'].items():
            print(name + ':', 'FOUND' if status['available'] else 'MISSING')
        print('Install missing modules into THIS Python using:')
        print('"' + result['python_executable'] + '" -m pip install <module-name>')
        print('NOTE: torch installation may require a GPU-specific wheel; see pytorch.org.')
        print('NOTE: module discovery does not guarantee import or GPU runtime success.')
    return 0 if result['all_modules_found'] and result['script_exists'] is not False else 1


if __name__ == '__main__':
    raise SystemExit(main())
