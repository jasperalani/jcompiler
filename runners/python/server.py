from flask import Flask, request, jsonify
import os
import sys
import time
import traceback
import concurrent.futures
import tempfile
import io
import contextlib
import signal
import RestrictedPython
from RestrictedPython import compile_restricted
from RestrictedPython import safe_globals

app = Flask(__name__)

@app.route('/run', methods=['POST'])
def run_code():
    start_time = time.time()

    if not request.is_json:
        return jsonify({
            'error': 'Request must be JSON',
            'stdout': '',
            'stderr': 'Request must be JSON',
            'exitCode': 1,
            'executionTime': 0
        }), 400

    data = request.get_json()
    code = data.get('code', '')
    timeout_seconds = data.get('timeout', 5)
    args = data.get('args', [])
    env_vars = data.get('env', {})

    if not code:
        return jsonify({
            'error': 'Code is required',
            'stdout': '',
            'stderr': 'Code is required',
            'exitCode': 1,
            'executionTime': 0
        }), 400

    # Get max execution time from environment or use default
    max_exec_time = int(os.environ.get('MAX_EXECUTION_TIME', 5))

    # Use the smaller of request timeout and max allowed timeout
    timeout_seconds = min(timeout_seconds, max_exec_time)

    result = run_with_timeout(execute_python_code, args=(code, timeout_seconds, args, env_vars), timeout_seconds=timeout_seconds)
    # result = execute_python_code(code, timeout_seconds, args, env_vars)

    # Add execution time
    result['executionTime'] = int((time.time() - start_time) * 1000)

    return jsonify(result)

@app.route('/health', methods=['GET'])
def health_check():
    return "OK", 200

def timeout_handler(signum, frame):
    raise TimeoutError("Code execution timed out")

def create_restricted_globals():
    """Create a restricted globals dictionary for RestrictedPython."""
    # restricted_globals = dict(globals())
    restricted_globals = dict(safe_globals)

    # Add print function that writes to our stdout
    stdout = []
    stderr = []

    def _print(*args, **kwargs):
        """Print function that captures output."""
        sep = kwargs.get('sep', ' ')
        end = kwargs.get('end', '\n')
        file = kwargs.get('file', None)

        output = sep.join(str(arg) for arg in args) + end

        if file is None or file == sys.stdout:
            stdout.append(output)
        else:  # Assume stderr
            stderr.append(output)

        return None

    restricted_globals['print'] = _print # type: ignore
    restricted_globals['_stdout'] = stdout # type: ignore
    restricted_globals['_stderr'] = stderr # type: ignore

    return restricted_globals

def execute_python_code(code, timeout_seconds, args, env_vars):
    """Execute Python code in a restricted environment."""
    original_argv = ""
    stdout = ""
    stderr = ""
    exit_code = 0
    error_msg = ""

    # Set environment variables
    original_env = os.environ.copy()
    try:
        for key, value in env_vars.items():
            os.environ[key] = str(value)

        # Set sys.argv
        original_argv = sys.argv
        sys.argv = ['script.py'] + args

        # Prepare stdout/stderr capture
        restricted_globals = create_restricted_globals()

        # todo: new timeout system
        # Set up timeout
        # signal.signal(signal.SIGALRM, timeout_handler)
        # signal.alarm(timeout_seconds)

        try:
            # Compile the code with RestrictedPython
            byte_code = compile(code, '<inline>', 'exec')
            # byte_code = compile_restricted(code, '<inline>', 'exec')

            # Execute the code
            exec(byte_code, restricted_globals)
            time.sleep(2)

            # Collect output
            stdout = ''.join(restricted_globals.get('_stdout', []))
            stderr = ''.join(restricted_globals.get('_stderr', []))

        except TimeoutError as e:
            stderr = f"Error: {str(e)}\n"
            exit_code = 1
            error_msg = "execution timed out"

        except Exception as e:
            stderr = f"Error: {str(e)}\n{traceback.format_exc()}"
            exit_code = 1
            error_msg = str(e)

    finally:
        # Restore environment and args
        os.environ.clear()
        os.environ.update(original_env)
        sys.argv = original_argv


    # print(stdout)
    stdout = stdout.rstrip("\n")

    return {
        'stdout': stdout,
        'stderr': stderr,
        'exitCode': exit_code,
        'error': error_msg
    }

def run_with_timeout(func, args=(), kwargs=None, timeout_seconds=5):
    if kwargs is None:
        kwargs = {}
    with concurrent.futures.ThreadPoolExecutor(max_workers=1) as executor:
        future = executor.submit(func, *args, **kwargs)
        try:
            return future.result(timeout=timeout_seconds)
        except concurrent.futures.TimeoutError:
            raise TimeoutError(f"Execution timed out after {timeout_seconds} seconds")

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8004)