const express = require('express');
const bodyParser = require('body-parser');
const { VM, VMScript } = require('vm2');
const fs = require('fs');
const path = require('path');
const { exec } = require('child_process');
const os = require('os');
const app = express();
const port = 8003;

// Parse JSON request bodies
app.use(bodyParser.json({ limit: '100kb' }));

app.post('/run', async (req, res) => {
	const { code, timeout, args, env } = req.body;
	
	if (!code) {
		return res.status(400).json({ error: 'Code is required' });
	}
	
	try {
		const result = await executeTypeScript(code, timeout, args, env);
		res.json(result);
	} catch (error) {
		res.status(500).json({
			error: error.message,
			stdout: '',
			stderr: error.message,
			exitCode: 1,
			executionTime: 0
		});
	}
});

app.get('/health', (req, res) => {
	res.status(200).send('OK');
});

function executeTypeScript(code, timeout, args = [], env = {}) {
	return new Promise(async (resolve) => {
		const startTime = Date.now();
		let stdout = '';
		let stderr = '';
		let exitCode = 0;
		
		// Get max execution time from environment or use default
		const maxExecTime = process.env.MAX_EXECUTION_TIME
		   ? parseInt(process.env.MAX_EXECUTION_TIME, 10)
		   : 20;
		
		// Use the smaller of request timeout and max allowed timeout
		//const timeoutMs = Math.min(
		//   timeout ? timeout * 1000 : timeoutMs,
		//   maxExecTime * 1000
		//);
		
		const timeoutMs = maxExecTime * 1000
		
		try {
			// Create temporary directory for TypeScript files
			//const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'ts-exec-'));
			const tmpDir = "/app/tmp"
			const tsFilePath = path.join(tmpDir, 'script.ts');
			const jsFilePath = path.join(tmpDir, 'script.js');
			
			if (!fs.existsSync(tmpDir)) {
				fs.mkdirSync(tmpDir);
			}
			
			// Write TypeScript code to file
			fs.writeFileSync(tsFilePath, code);
			
			// Transpile TypeScript to JavaScript
			await new Promise((resolveCompile, rejectCompile) => {
				const compileProcess = exec(`npx tsc ${tsFilePath} --target ES2020 --module commonjs --outDir ${tmpDir}`,
				   { timeout: 30000 },
				   (error, stdout, stderr) => {
					   if (error) {
						   rejectCompile({
							   stdout: '',
							   stderr: `TypeScript compilation error: ${stderr || error.message}`,
							   exitCode: 1,
							   executionTime: Date.now() - startTime,
							   error: error.message
						   });
					   } else {
						   resolveCompile();
					   }
				   }
				);
			});
			
			// If compilation successful, read the JS file
			const jsCode = fs.readFileSync(jsFilePath, 'utf8');
			
			// Create console.log interceptor to capture stdout
			const customConsole = {
				log: (...args) => {
					stdout += args.map(arg =>
					   typeof arg === 'object' ? JSON.stringify(arg) : String(arg)
					).join(' ') + '\n';
				},
				error: (...args) => {
					stderr += args.map(arg =>
					   typeof arg === 'object' ? JSON.stringify(arg) : String(arg)
					).join(' ') + '\n';
				},
				warn: (...args) => {
					stderr += args.map(arg =>
					   typeof arg === 'object' ? JSON.stringify(arg) : String(arg)
					).join(' ') + '\n';
				},
				info: (...args) => {
					stdout += args.map(arg =>
					   typeof arg === 'object' ? JSON.stringify(arg) : String(arg)
					).join(' ') + '\n';
				}
			};
			
			// Execute the transpiled JavaScript
			const vm = new VM({
				timeout: timeoutMs,
				sandbox: {
					console: customConsole,
					process: {
						argv: ['node', 'script.js', ...args],
						env: { ...env }
					},
					setTimeout,
					clearTimeout,
					setInterval,
					clearInterval,
					Buffer,
				}
			});
			
			const script = new VMScript(jsCode);
			vm.run(script);
			
			//stdout = stdout.substring(0, stdout.length - 1);
			
			// Clean up temporary directory
			fs.rmSync(tmpDir, { recursive: true, force: true });
			
			resolve({
				stdout,
				stderr,
				exitCode,
				executionTime: Date.now() - startTime,
				error: ''
			});
		} catch (error) {
			if (error.stdout !== undefined) {
				// This is an error object from the TypeScript compilation
				resolve(error);
			} else if (error.message && error.message.includes('Script execution timed out')) {
				resolve({
					stdout,
					stderr: stderr + 'Error: Script execution timed out\n',
					exitCode: 1,
					executionTime: timeoutMs,
					error: 'execution timed out'
				});
			} else {
				resolve({
					stdout,
					stderr: stderr + `Error: ${error.message}\n`,
					exitCode: 1,
					executionTime: Date.now() - startTime,
					error: error.message
				});
			}
		}
	});
}

app.listen(port, () => {
	console.log(`TypeScript runner listening at http://localhost:${port}`);
});