import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';

const require = createRequire(import.meta.url);
const { name, version } = require('@roxi/routify/package.json');

const rootDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const pagesDir = path.join(rootDir, 'frontend/pages');
const routifyDir = path.join(rootDir, '.routify');
const extensions = ['svelte'];

const defaultNode = {
  isDir: false,
  ext: 'svelte',
  isLayout: false,
  isReset: false,
  isIndex: false,
  isFallback: false,
  isPage: false,
  ownMeta: {},
  meta: {
    recursive: true,
    preload: false,
    prerender: true,
  },
  id: '__fallback',
};

const devProps = [
  'file',
  'filepath',
  'name',
  'badExt',
  'relativeDir',
  'absolutePath',
  'importPath',
  'isFile',
];

function makeLegalIdentifier(str) {
  let id = str.replace(/[^a-zA-Z0-9_$]/g, '_');
  if (/^\d/.test(id)) id = `_${id}`;
  return id.replace(/_$/, '') || '_';
}

function matchExtension(filename) {
  for (const ext of extensions) {
    const suffix = `.${ext}`;
    if (filename.endsWith(suffix)) {
      return [filename.slice(0, -suffix.length), ext, false];
    }
  }
  const [basename, ext] = filename.split('.');
  return [basename, ext, true];
}

async function buildFileTree(absoluteDir, relativeDir = '') {
  const entries = await fs.readdir(absoluteDir);
  const children = [];

  for (const filename of entries) {
    const absolutePath = path.join(absoluteDir, filename);
    const filepath = `${relativeDir}/${filename}`;
    const stat = await fs.stat(absolutePath);

    if (stat.isDirectory()) {
      if (filename.startsWith('_')) continue;
      children.push({
        isDir: true,
        ext: '',
        children: await buildFileTree(absolutePath, filepath),
        path: filepath.replace(/\[([^\]]+)\]/g, ':$1'),
      });
      continue;
    }

    const [name, ext, badExt] = matchExtension(filename);
    if (badExt) continue;

    const importPath = path
      .relative(routifyDir, absolutePath)
      .replace(/\\/g, '/');

    let routePath = filepath.slice(0, -(ext.length + 1));
    routePath = routePath.replace(/\[([^\]]+)\]/g, ':$1');

    const isIndex = name === 'index';
    const node = {
      isPage: true,
      path: routePath,
      id: makeLegalIdentifier(routePath),
      importPath,
      component: `'''() => ${makeLegalIdentifier(routePath)}'''`,
    };

    if (isIndex) node.isIndex = true;
    children.push(node);
  }

  children.sort((a, b) => a.path.localeCompare(b.path, undefined, { numeric: true }));
  return children;
}

function stripDefaultsAndDevProps(node) {
  const strippedNode = {};

  for (const [key, value] of Object.entries(node)) {
    if (devProps.includes(key)) continue;
    if (JSON.stringify(defaultNode[key]) === JSON.stringify(value)) continue;
    strippedNode[key] =
      key === 'children' ? value.map(stripDefaultsAndDevProps) : value;
  }

  return strippedNode;
}

function escapeTemplateJson(tree) {
  return JSON.stringify(tree, null, 2).replace(/"'''|'''"/g, '');
}

async function writeRoutes() {
  const importBasePath = path
    .relative(routifyDir, pagesDir)
    .replace(/\\/g, '/');

  const tree = {
    root: true,
    children: await buildFileTree(pagesDir),
    path: '/',
  };

  const imports = [];
  collectImports(tree.children, imports);

  const template = `
/**
 * ${name} ${version}
 * File generated ${new Date()}
 */

export const __version = "${version}"
export const __timestamp = "${new Date().toISOString()}"

//buildRoutes
import { buildClientTree } from "@roxi/routify/runtime/buildRoutes"

//imports
${imports.map((route) => `import ${route.id} from '${route.importPath}'`).join('\n')}

//options
export const options = {}

//tree
export const _tree = ${escapeTemplateJson(stripDefaultsAndDevProps(tree))}


export const {tree, routes} = buildClientTree(_tree)

`;

  await fs.mkdir(routifyDir, { recursive: true });
  await fs.writeFile(path.join(routifyDir, 'routes.js'), template, 'utf8');
  await fs.writeFile(
    path.join(routifyDir, 'config.js'),
    `module.exports = ${JSON.stringify(
      {
        pages: './frontend/pages',
        sourceDir: 'public',
        routifyDir: '.routify',
        ignore: '',
        dynamicImports: false,
        singleBuild: true,
        noHashScroll: false,
        distDir: 'dist',
        hashScroll: true,
        extensions: ['html', 'svelte', 'md', 'svx'],
        started: new Date().toISOString(),
      },
      null,
      2,
    )}\n`,
    'utf8',
  );
}

function collectImports(children, imports = []) {
  for (const child of children) {
    if (child.children) collectImports(child.children, imports);
    if (child.isPage) imports.push(child);
  }
  return imports;
}

const watch = process.argv.includes('--watch');

await writeRoutes();

if (watch) {
  const watcher = fs.watch(pagesDir, { recursive: true }, async () => {
    await writeRoutes();
  });

  process.on('SIGINT', () => {
    watcher.close();
    process.exit(0);
  });
}
