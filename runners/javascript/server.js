const express = require('express');
const bodyParser = require('body-parser');
const { VM, VMScript } = require('vm2');
const app = express();
const port = 8082;

// Parse JSON request bodies
app.use(bodyParser.json({ limit: '100kb' }));

app.post('/run', async (req, res) => {
	const { code, timeout, args, env } = req.body;
	
	if (!code) {
		return res.status(400).json({ error: 'Code is required' });
	}
	
	try {
		const result = await executeJavaScript(code, timeout, args, env);
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

function executeJavaScript(code, timeout, args = [], env = {}) {
	return new Promise((resolve) => {
		const startTime = Date.now();
		let stdout = '';
		let stderr = '';
		let exitCode = 0;
		
		// Get max execution time from environment or use default
		const maxExecTime = process.env.MAX_EXECUTION_TIME
		   ? parseInt(process.env.MAX_EXECUTION_TIME, 10)
		   : 5;
		
		// Use the smaller of request timeout and max allowed timeout
		const timeoutMs = Math.min(
		   timeout ? timeout * 1000 : 5000,
		   maxExecTime * 1000
		);
		
		try {
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
			
			// Create sandbox with safe defaults
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
			
			// Execute the code
			const script = new VMScript(code);
			vm.run(script);
			
			resolve({
				stdout,
				stderr,
				exitCode,
				executionTime: Date.now() - startTime,
				error: ''
			});
		} catch (error) {
			if (error.message.includes('Script execution timed out')) {
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
	console.log(`JavaScript runner listening at http://localhost:${port}`);
});