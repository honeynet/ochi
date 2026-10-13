import svelte from 'rollup-plugin-svelte';
import commonjs from '@rollup/plugin-commonjs';
import resolve from '@rollup/plugin-node-resolve';
import livereload from 'rollup-plugin-livereload';
import terser from '@rollup/plugin-terser';
import sveltePreprocess from 'svelte-preprocess';
import typescript from '@rollup/plugin-typescript';

const production = !process.env.ROLLUP_WATCH;

function serve() {
	let server;

	function toExit() {
		if (server) server.kill(0);
	}

	return {
		writeBundle() {
			if (server) return;
			server = require('child_process').spawn('npm', ['run', 'start', '--', '--dev'], {
				stdio: ['ignore', 'inherit', 'inherit'],
				shell: true
			});

			process.on('SIGTERM', toExit);
			process.on('exit', toExit);
		}
	};
}

// Collects component CSS into one file. Unlike rollup-plugin-css-only, this also follows
// dynamic imports, which is how Routify 3 loads pages and modules.
function css({ output }) {
	const styles = new Map();

	return {
		name: 'css',
		transform(code, id) {
			if (!id.endsWith('.css')) return;
			styles.set(id, code);
			return { code: '', map: { mappings: '' } };
		},
		generateBundle(_, bundle) {
			const seen = new Set();
			const ordered = [];
			const visit = (id) => {
				if (seen.has(id)) return;
				seen.add(id);
				const info = this.getModuleInfo(id);
				if (!info) return;
				[...info.importedIds, ...info.dynamicallyImportedIds].forEach(visit);
				if (styles.has(id)) ordered.push(styles.get(id));
			};
			for (const chunk of Object.values(bundle)) {
				if (chunk.type === 'chunk' && chunk.isEntry) visit(chunk.facadeModuleId);
			}
			this.emitFile({ type: 'asset', fileName: output, source: ordered.join('\n') });
		}
	};
}

function silence(warning, warn) {
	if (warning.code == 'CIRCULAR_DEPENDENCY') {
		// Upstream packages with known internal cycles (not app code).
		if (/node_modules\/(svelte|chevrotain|@roxi\/routify)/.test(warning.message || warning.importer || '')) return;
	}
	if (/node_modules.*?chevrotain/.test(warning.id)) {
		if (warning.code == 'EVAL') return;
		if (warning.code == 'THIS_IS_UNDEFINED') return;
		return;
	}
	if (warning.code == 'THIS_IS_UNDEFINED' && /node_modules.*?(@roxi\/routify|consolite)/.test(warning.id)) return;
	warn(warning);
}

export default {
	input: 'frontend/main.ts',
	output: {
		sourcemap: !production,
		format: 'iife',
		name: 'app',
		file: 'public/build/bundle.js',
		// Routify 3 lazy-loads pages via import(); keep a single IIFE bundle
		inlineDynamicImports: true,
	},
	onwarn: silence,
	plugins: [
		svelte({
			preprocess: sveltePreprocess({ sourceMap: !production }),
			compilerOptions: {
				// enable run-time checks when not in production
				dev: !production
			}
		}),
		// we'll extract any component CSS out into
		// a separate file - better for performance
		css({ output: 'bundle.css' }),

		// If you have external dependencies installed from
		// npm, you'll most likely need these plugins. In
		// some cases you'll need additional configuration -
		// consult the documentation for details:
		// https://github.com/rollup/plugins/tree/master/packages/commonjs
		resolve({
			browser: true,
			dedupe: ['svelte']
		}),
		commonjs(),
		typescript({
			sourceMap: !production,
			inlineSources: !production
		}),

		// In dev mode, call `npm run start` once
		// the bundle has been generated
		!production && serve(),

		// Watch the `public` directory and refresh the
		// browser on changes when not in production
		!production && livereload('public'),

		// If we're building for production (npm run build
		// instead of npm run dev), minify
		production && terser()
	],
	watch: {
		clearScreen: false
	}
};
