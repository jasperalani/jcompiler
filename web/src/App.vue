<template>
  <div class="container">
    <h1>Online Code Compiler</h1>

    <div class="compiler-container">
      <div class="language-selector">
        <label for="language">Select Language:</label>
        <select id="language" v-model="selectedLanguage">
          <option v-for="lang in languages" :key="lang.value" :value="lang.value">
            {{ lang.name }}
          </option>
        </select>
        <button @click="runCode" :disabled="isRunning">
          {{ isRunning ? 'Running...' : 'Run Code' }}
        </button>
      </div>

      <div class="editor-container">
        <div class="code-editor">
          <h2>Code</h2>
          <textarea
            v-model="code"
            placeholder="Write your code here..."
            @keydown.tab.prevent="insertTab"
          ></textarea>
        </div>

        <div class="code-output">
          <h2>Output</h2>
          <textarea
            v-model="output"
            placeholder="Your output will appear here..."
            readonly
          ></textarea>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref } from 'vue';
import axios from 'axios';

interface Language {
  name: string;
  value: string;
  example: string;
}

export default defineComponent({
  name: 'App',
  setup() {
    const languages = ref<Language[]>([
      {
        name: 'JavaScript',
        value: 'js',
        example: 'console.log("Hello from JavaScript!");'
      },
      {
        name: 'TypeScript',
        value: 'ts',
        example: 'const message: string = "Hello from TypeScript!";\nconsole.log(message);'
      },
      {
        name: 'Python',
        value: 'python',
        example: 'print("Hello from Python!")'
      },
      {
        name: 'Go',
        value: 'go',
        example: 'package main\n\nimport "fmt"\n\nfunc main() {\n\tfmt.Println("Hello from Go!")\n}'
      }
    ]);

    const selectedLanguage = ref<string>(languages.value[0].value);
    const code = ref<string>(languages.value[0].example);
    const output = ref<string>('');
    const isRunning = ref<boolean>(false);

    // Update example code when language changes
    const updateExampleCode = () => {
      const selected = languages.value.find(lang => lang.value === selectedLanguage.value);
      if (selected) {
        code.value = selected.example;
      }
    };

    // Watch for language changes
    const watchLanguage = () => {
      updateExampleCode();
    };

    // Run code (mock implementation)
    const runCode = async () => {
      isRunning.value = true;
      output.value = 'Running code...';

      try {
        const data = {
          code: btoa(code.value),
          language: selectedLanguage.value
        }

        try {
          const response = await axios.post('http://jcompiler.riseandflour.co.uk/api/process', data, {
            headers: {
              'Content-Type': 'application/json',
            },
          });
          console.log('Response:', response.data);

          let serverOutput = "Server error"
          if(response.data.stdout !== ''){
            serverOutput = response.data.stdout;
          }

          output.value = serverOutput

        } catch (error) {
          console.error('Error:', error);
        }
      } catch (error) {
        output.value = `Error: ${error}`;
      } finally {
        isRunning.value = false;
      }
    };

    // Helper to insert tabs in textarea
    const insertTab = (event: KeyboardEvent) => {
      const target = event.target as HTMLTextAreaElement;
      const start = target.selectionStart;
      const end = target.selectionEnd;

      code.value = code.value.substring(0, start) + '\t' + code.value.substring(end);

      // Set cursor position after the inserted tab
      setTimeout(() => {
        target.selectionStart = target.selectionEnd = start + 1;
      }, 0);
    };

    return {
      languages,
      selectedLanguage,
      code,
      output,
      runCode,
      insertTab,
      isRunning,
      watchLanguage
    };
  },
  watch: {
    selectedLanguage() {
      this.watchLanguage();
    }
  }
});
</script>

<style scoped>
.container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
  font-family: 'Arial', sans-serif;
}

h1 {
  text-align: center;
  margin-bottom: 2rem;
  color: #333;
}

h2 {
  color: #555;
  margin-bottom: 0.5rem;
}

.compiler-container {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.language-selector {
  display: flex;
  gap: 1rem;
  align-items: center;
  margin-bottom: 1rem;
}

select {
  padding: 0.5rem;
  border-radius: 4px;
  border: 1px solid #ccc;
  font-size: 1rem;
}

button {
  padding: 0.5rem 1rem;
  background-color: #4CAF50;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 1rem;
  font-weight: bold;
  transition: background-color 0.3s;
}

button:hover {
  background-color: #45a049;
}

button:disabled {
  background-color: #cccccc;
  cursor: not-allowed;
}

.editor-container {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
  height: 70vh;
}

.code-editor, .code-output {
  display: flex;
  flex-direction: column;
  height: 100%;
}

textarea {
  flex-grow: 1;
  padding: 1rem;
  font-family: 'Courier New', monospace;
  font-size: 1rem;
  line-height: 1.5;
  border: 1px solid #ccc;
  border-radius: 4px;
  resize: none;
  background-color: #f9f9f9;
}

.code-editor textarea {
  background-color: #1e1e1e;
  color: #d4d4d4;
}

.code-output textarea {
  background-color: #2d2d2d;
  color: #e7e7e7;
}

@media (max-width: 768px) {
  .editor-container {
    grid-template-columns: 1fr;
    height: auto;
  }

  .code-editor, .code-output {
    height: 50vh;
  }
}
</style>