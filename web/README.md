# Online Code Compiler

A modern, responsive frontend for an online code compiler built with Vue 3, TypeScript, and Vite. This web application allows users to write and execute code in multiple programming languages directly in the browser.

![Online Code Compiler Screenshot](https://via.placeholder.com/800x450.png?text=Online+Code+Compiler)

## Getting Started

### Prerequisites

- Node.js (v14.0.0 or higher)
- npm (v6.0.0 or higher)

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/code-compiler-frontend.git
   cd code-compiler-frontend
   ```

2. Install dependencies:
   ```bash
   npm install
   ```

3. Start the development server:
   ```bash
   npm run dev
   ```

4. Open your browser and navigate to `http://localhost:5173`

## Backend Integration

The current implementation includes a mock backend for demonstration purposes. To connect to a real code execution backend:

1. Modify the `runCode` function in `App.vue`:
   ```typescript
   const runCode = async () => {
     isRunning.value = true;
     output.value = 'Running code...';
     
     try {
       const response = await fetch('YOUR_BACKEND_API_URL', {
         method: 'POST',
         headers: {
           'Content-Type': 'application/json',
         },
         body: JSON.stringify({
           language: selectedLanguage.value,
           code: code.value
         }),
       });
       
       const data = await response.json();
       output.value = data.output;
     } catch (error) {
       output.value = `Error: ${error}`;
     } finally {
       isRunning.value = false;
     }
   };
   ```

## Building for Production

1. Build the project:
   ```bash
   npm run build
   ```

2. Preview the production build:
   ```bash
   npm run preview
   ```

3. Deploy the contents of the `dist` folder to your hosting provider

## Future Enhancements

- Add more programming languages
- Implement syntax highlighting
- Add ability to save and share code snippets
- Implement user authentication
- Add code formatting options
- Create a split-view mode for code and output

## Contributing

1. Fork the repository
2. Create your feature branch: `git checkout -b feature/amazing-feature`
3. Commit your changes: `git commit -m 'Add some amazing feature'`
4. Push to the branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- [Vue.js](https://vuejs.org/)
- [Vite](https://vitejs.dev/)
- [TypeScript](https://www.typescriptlang.org/)