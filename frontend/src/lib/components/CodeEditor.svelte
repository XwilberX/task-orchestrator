<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import loader from '@monaco-editor/loader';

	let {
		value = $bindable(''),
		language = 'python',
		readonly = false,
		height = '400px'
	}: {
		value?: string;
		language?: string;
		readonly?: boolean;
		height?: string;
	} = $props();

	const LANG_MAP: Record<string, string> = {
		python: 'python',
		nodejs: 'javascript',
		typescript: 'typescript',
		go: 'go',
		java: 'java'
	};

	let container: HTMLDivElement | undefined;
	let editor: import('monaco-editor').editor.IStandaloneCodeEditor | null = null;
	let monaco: typeof import('monaco-editor') | null = null;

	onMount(async () => {
		monaco = await loader.init();

		// Tema oscuro custom alineado con el design system
		if (!monaco) return;
		monaco.editor.defineTheme('orchestrator-dark', {
			base: 'vs-dark',
			inherit: true,
			rules: [
				{ token: 'comment', foreground: '6b7280', fontStyle: 'italic' },
				{ token: 'keyword', foreground: 'd0bcff' },
				{ token: 'string', foreground: '86efac' },
				{ token: 'number', foreground: 'fb923c' },
			],
			colors: {
				'editor.background': '#080809',
				'editor.foreground': '#e7e4ec',
				'editorLineNumber.foreground': '#3d3b3e',
				'editorLineNumber.activeForeground': '#75757c',
				'editor.selectionBackground': '#5516be40',
				'editor.lineHighlightBackground': '#19191d',
				'editorCursor.foreground': '#d0bcff',
				'editor.inactiveSelectionBackground': '#2b2c3240',
			}
		});

		if (!container) return;

		editor = monaco.editor.create(container, {
			value,
			language: LANG_MAP[language] ?? language,
			theme: 'orchestrator-dark',
			readOnly: readonly,
			fontSize: 13,
			fontFamily: "'JetBrains Mono Variable', 'JetBrains Mono', monospace",
			fontLigatures: true,
			lineNumbers: 'on',
			minimap: { enabled: false },
			scrollBeyondLastLine: false,
			wordWrap: 'on',
			padding: { top: 12, bottom: 12 },
			renderLineHighlight: 'line',
			smoothScrolling: true,
			cursorBlinking: 'smooth',
			automaticLayout: true,
			tabSize: 4,
			insertSpaces: true,
		});

		editor.onDidChangeModelContent(() => {
			value = editor?.getValue() ?? '';
		});
	});

	// Cambiar lenguaje cuando cambia el runtime
	$effect(() => {
		if (!editor || !monaco) return;
		const m = monaco;
		const lang = LANG_MAP[language] ?? language;
		const model = editor.getModel();
		if (model) {
			m.editor.setModelLanguage(model, lang);
		}
	});

	// Sincronizar value externo → editor (solo si difiere)
	$effect(() => {
		if (!editor) return;
		if (editor.getValue() !== value) {
			editor.setValue(value);
		}
	});

	onDestroy(() => {
		editor?.dispose();
	});
</script>

<div
	bind:this={container}
	style="height: {height}"
	class="w-full overflow-hidden rounded-lg border border-[#1f1f24]"
></div>
