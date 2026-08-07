// Server-authoritative navigation state for the currently-live terminal.
// All connected players share one position (folder path / open entry /
// active command) so that when one player enters a folder, everyone sees it.

function defaultNav() {
  return { path: ['root'], mode: 'list', viewEntryId: null, commandNodeId: null };
}

function findNodeById(root, id) {
  if (!root) return null;
  if (root.id === id) return root;
  if (root.children) {
    for (const c of root.children) {
      const found = findNodeById(c, id);
      if (found) return found;
    }
  }
  return null;
}

function currentFolderFromPath(tree, path) {
  let cur = tree;
  for (let i = 1; i < path.length; i++) {
    const next = cur.children && cur.children.find(c => c.id === path[i]);
    if (!next) return cur;
    cur = next;
  }
  return cur;
}

function revalidateNavPath(tree, path) {
  const valid = ['root'];
  let cur = tree;
  for (let i = 1; i < path.length; i++) {
    const node = cur.children && cur.children.find(c => c.id === path[i] && c.type === 'folder');
    if (!node) break;
    valid.push(path[i]);
    cur = node;
  }
  return valid;
}

// Mutates and returns `nav` in response to a client action.
function applyNavAction(nav, tree, action) {
  if (action.action === 'enter') {
    const folder = currentFolderFromPath(tree, nav.path);
    const node = folder.children && folder.children.find(c => c.id === action.nodeId && c.type === 'folder');
    if (!node) return nav;
    nav.path = [...nav.path, node.id];
    nav.mode = 'list';
    nav.viewEntryId = null;
    nav.commandNodeId = null;
  } else if (action.action === 'back') {
    if (nav.mode === 'entry') {
      nav.mode = 'list';
      nav.viewEntryId = null;
    } else if (nav.path.length > 1) {
      nav.path = nav.path.slice(0, -1);
      nav.commandNodeId = null;
    }
  } else if (action.action === 'command') {
    const folder = currentFolderFromPath(tree, nav.path);
    const node = folder.children && folder.children.find(c => c.id === action.nodeId && c.type === 'command');
    if (!node) return nav;
    nav.commandNodeId = node.id;
  } else if (action.action === 'entry') {
    const folder = currentFolderFromPath(tree, nav.path);
    const node = folder.children && folder.children.find(c => c.id === action.nodeId && c.type === 'entry');
    if (!node) return nav;
    nav.mode = 'entry';
    nav.viewEntryId = node.id;
    nav.commandNodeId = null;
  }
  return nav;
}

// Re-checks a nav position against a (possibly just-edited) tree, dropping
// anything that no longer exists. Used after the GM publishes tree changes.
function revalidateNav(nav, tree) {
  const path = revalidateNavPath(tree, nav.path);
  const folder = currentFolderFromPath(tree, path);

  let mode = nav.mode;
  let viewEntryId = nav.viewEntryId;
  let commandNodeId = nav.commandNodeId;

  if (mode === 'entry') {
    const node = findNodeById(tree, viewEntryId);
    if (!node || node.type !== 'entry') {
      mode = 'list';
      viewEntryId = null;
    }
  }
  if (commandNodeId) {
    const stillThere = folder.children && folder.children.find(c => c.id === commandNodeId && c.type === 'command');
    if (!stillThere) commandNodeId = null;
  }

  return { path, mode, viewEntryId, commandNodeId };
}

module.exports = { defaultNav, applyNavAction, revalidateNav };
