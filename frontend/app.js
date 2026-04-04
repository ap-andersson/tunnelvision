// TunnelVision frontend — Wails + Vanilla JS

// ── State ──────────────────────────────────────────────
let selectedType = null;   // 'tunnel' | 'folder' | 'logs' | null
let selectedId = null;     // filename (tunnel) or folder name
let currentStatus = { connected: false, activeTunnel: '', activeInterfaces: [] };
let expandedFolders = new Set(); // track which folders are open
let logAutoFollow = true;  // auto-scroll log view to bottom
let logAllLines = [];      // all log lines (unfiltered)
let logFilterText = '';    // current filter string

// ── DOM refs ───────────────────────────────────────────
const $tree = document.getElementById('tunnel-tree');
const $tunnelView = document.getElementById('tunnel-view');
const $folderView = document.getElementById('folder-view');
const $emptyView = document.getElementById('empty-view');
const $logView = document.getElementById('log-view');
const $logOutput = document.getElementById('log-output');
const $logContainer = document.getElementById('log-container');
const $btnFollow = document.getElementById('btn-follow');
const $btnLogs = document.getElementById('btn-logs');
const $logFilter = document.getElementById('log-filter');
const $editor = document.getElementById('config-editor');
const $infoAddress = document.getElementById('info-address');
const $infoEndpoint = document.getElementById('info-endpoint');
const $infoDNS = document.getElementById('info-dns');
const $folderName = document.getElementById('folder-name');
const $folderCount = document.getElementById('folder-tunnel-count');
const $folderSelect = document.getElementById('folder-select');
const $btnDisconnect = document.getElementById('btn-disconnect');
const $btnDelete = document.getElementById('btn-delete');

// ── Wails bindings (generated at runtime) ──────────────
// These become available as window.go.main.TunnelService.MethodName()
// and window.go.main.ConfigService.MethodName()

function tunnelSvc() { return window.go.main.TunnelService; }
function configSvc() { return window.go.main.ConfigService; }
function logSvc() { return window.go.main.LogService; }

// ── Initialization ─────────────────────────────────────
document.addEventListener('DOMContentLoaded', async () => {
    await refreshAll();
    bindToolbar();
    bindEditorActions();
    bindDialogs();
    bindLogs();

    // Listen for backend events
    window.runtime.EventsOn('tunnel:state-changed', async () => {
        await refreshStatus();
        await refreshTree();
        updateToolbarState();
    });

    window.runtime.EventsOn('log:line', (line) => {
        appendLogLine(line);
    });
});

// ── Refresh logic ──────────────────────────────────────
async function refreshAll() {
    await refreshStatus();
    await refreshTree();
    updateToolbarState();
}

async function refreshStatus() {
    try {
        currentStatus = await tunnelSvc().GetStatus();
    } catch (e) {
        console.error('Failed to get status:', e);
    }
}

async function refreshTree() {
    try {
        const meta = await configSvc().GetMetadata();
        renderTree(meta);
    } catch (e) {
        console.error('Failed to get metadata:', e);
    }
}

// ── Tree rendering ─────────────────────────────────────
function renderTree(meta) {
    $tree.innerHTML = '';

    // Determine which folder contains the active tunnel
    let activeFolderName = null;
    if (currentStatus.connected && currentStatus.activeTunnel) {
        for (const [folder, tunnels] of Object.entries(meta.folders)) {
            if (tunnels.includes(currentStatus.activeTunnel)) {
                activeFolderName = folder;
                break;
            }
        }
    }

    // Folders
    const folderNames = Object.keys(meta.folders).sort();
    if (folderNames.length > 0) {
        const groupsLabel = document.createElement('div');
        groupsLabel.className = 'tree-section-label';
        groupsLabel.textContent = 'Groups';
        $tree.appendChild(groupsLabel);
    }
    for (const folder of folderNames) {
        const tunnels = meta.folders[folder] || [];
        const wrapper = document.createElement('div');
        wrapper.className = 'tree-folder';

        const isOpen = expandedFolders.has(folder);
        if (isOpen) {
            wrapper.classList.add('open');
        }

        // Mark folder as active if it contains the connected tunnel
        if (activeFolderName === folder) {
            wrapper.classList.add('active-folder');
        }

        const header = document.createElement('div');
        header.className = 'tree-folder-header';
        header.dataset.type = 'folder';
        header.dataset.id = folder;
        if (selectedType === 'folder' && selectedId === folder) {
            header.classList.add('selected');
        }

        // Folder label
        const label = document.createElement('span');
        label.className = 'folder-label';
        label.textContent = `${folder} (${tunnels.length})`;

        // Expand toggle
        const toggle = document.createElement('span');
        toggle.className = 'folder-toggle';
        toggle.addEventListener('click', (e) => {
            e.stopPropagation();
            if (wrapper.classList.contains('open')) {
                wrapper.classList.remove('open');
                expandedFolders.delete(folder);
            } else {
                wrapper.classList.add('open');
                expandedFolders.add(folder);
            }
        });

        header.appendChild(label);
        header.appendChild(toggle);
        // Clicking the header selects the folder
        header.addEventListener('click', () => {
            selectItem('folder', folder);
        });
        wrapper.appendChild(header);

        const tunnelContainer = document.createElement('div');
        tunnelContainer.className = 'folder-tunnels';
        for (const filename of tunnels) {
            tunnelContainer.appendChild(createTunnelItem(filename));
        }
        wrapper.appendChild(tunnelContainer);
        $tree.appendChild(wrapper);
    }

    // Ungrouped tunnels
    if (meta.ungrouped && meta.ungrouped.length > 0) {
        if (folderNames.length > 0) {
            const label = document.createElement('div');
            label.className = 'tree-section-label';
            label.textContent = 'Ungrouped';
            $tree.appendChild(label);
        }
        for (const filename of meta.ungrouped) {
            $tree.appendChild(createTunnelItem(filename));
        }
    }

    // Update folder select dropdown
    updateFolderSelect(folderNames);
}

function createTunnelItem(filename) {
    const div = document.createElement('div');
    div.className = 'tree-item';
    div.dataset.type = 'tunnel';
    div.dataset.id = filename;

    const name = filename.replace(/\.conf$/, '');
    div.textContent = name;

    if (currentStatus.activeTunnel === filename) {
        div.classList.add('active-tunnel');
    }
    if (selectedType === 'tunnel' && selectedId === filename) {
        div.classList.add('selected');
    }

    div.addEventListener('click', () => selectItem('tunnel', filename));
    return div;
}

function updateFolderSelect(folderNames) {
    $folderSelect.innerHTML = '<option value="">Move to...</option>';
    $folderSelect.innerHTML += '<option value="">(Ungrouped)</option>';
    for (const name of folderNames) {
        const opt = document.createElement('option');
        opt.value = name;
        opt.textContent = name;
        $folderSelect.appendChild(opt);
    }
}

// ── Selection handling ─────────────────────────────────
async function selectItem(type_, id) {
    selectedType = type_;
    selectedId = id;

    // Update tree highlighting
    document.querySelectorAll('.tree-item.selected, .tree-folder-header.selected').forEach(el => {
        el.classList.remove('selected');
    });
    $btnLogs.classList.remove('selected');

    if (type_ === 'logs') {
        $btnLogs.classList.add('selected');
    } else {
        const sel = document.querySelector(`[data-type="${type_}"][data-id="${CSS.escape(id)}"]`);
        if (sel) sel.classList.add('selected');
    }

    updateToolbarState();

    if (type_ === 'tunnel') {
        await showTunnelView(id);
    } else if (type_ === 'folder') {
        showFolderView(id);
    } else if (type_ === 'logs') {
        await showLogView();
    }
}

async function showTunnelView(filename) {
    $tunnelView.classList.remove('hidden');
    $folderView.classList.add('hidden');
    $emptyView.classList.add('hidden');
    $logView.classList.add('hidden');

    try {
        const cfg = await configSvc().ReadConfig(filename);
        $editor.value = cfg.raw;
        $infoAddress.textContent = cfg.address ? `Address: ${cfg.address}` : '';
        $infoEndpoint.textContent = cfg.endpoint ? `Endpoint: ${cfg.endpoint}` : '';
        $infoDNS.textContent = cfg.dns ? `DNS: ${cfg.dns}` : '';
    } catch (e) {
        showError('Failed to read config: ' + e);
    }
}

async function showFolderView(folder) {
    $tunnelView.classList.add('hidden');
    $folderView.classList.remove('hidden');
    $emptyView.classList.add('hidden');
    $logView.classList.add('hidden');

    $folderName.textContent = folder;
    try {
        const meta = await configSvc().GetMetadata();
        const tunnels = meta.folders[folder] || [];
        $folderCount.textContent = `${tunnels.length} tunnel${tunnels.length !== 1 ? 's' : ''}`;
    } catch (e) {
        $folderCount.textContent = '';
    }
}

function showEmptyView() {
    $tunnelView.classList.add('hidden');
    $folderView.classList.add('hidden');
    $emptyView.classList.remove('hidden');
    $logView.classList.add('hidden');
    $btnLogs.classList.remove('selected');
    selectedType = null;
    selectedId = null;
    updateToolbarState();
}

// ── Toolbar state ──────────────────────────────────────
function updateToolbarState() {
    const hasTunnelSelected = selectedType === 'tunnel';
    const hasFolderSelected = selectedType === 'folder';
    const hasSelection = hasTunnelSelected || hasFolderSelected;

    $btnDelete.disabled = !hasSelection;
    $btnDisconnect.disabled = !currentStatus.connected;
}

// ── Log view ───────────────────────────────────────────
function bindLogs() {
    $btnLogs.addEventListener('click', () => {
        selectItem('logs', null);
    });

    $logContainer.addEventListener('scroll', () => {
        const el = $logContainer;
        const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 30;
        if (atBottom && !logAutoFollow) {
            logAutoFollow = true;
            $btnFollow.classList.add('hidden');
        } else if (!atBottom && logAutoFollow) {
            logAutoFollow = false;
            $btnFollow.classList.remove('hidden');
        }
    });

    $btnFollow.addEventListener('click', () => {
        logAutoFollow = true;
        $btnFollow.classList.add('hidden');
        $logContainer.scrollTop = $logContainer.scrollHeight;
    });

    document.getElementById('btn-clear-logs').addEventListener('click', () => {
        logAllLines = [];
        $logOutput.textContent = '';
    });

    $logFilter.addEventListener('input', () => {
        logFilterText = $logFilter.value.toLowerCase();
        renderFilteredLogs();
    });
}

async function showLogView() {
    $tunnelView.classList.add('hidden');
    $folderView.classList.add('hidden');
    $emptyView.classList.add('hidden');
    $logView.classList.remove('hidden');

    try {
        const lines = await logSvc().GetLogs();
        logAllLines = lines;
    } catch (e) {
        logAllLines = ['Failed to load logs: ' + e];
    }

    renderFilteredLogs();

    // Scroll to bottom and enable auto-follow
    logAutoFollow = true;
    $btnFollow.classList.add('hidden');
    $logContainer.scrollTop = $logContainer.scrollHeight;
}

function appendLogLine(line) {
    logAllLines.push(line);

    // Trim stored lines to prevent unbounded memory growth
    if (logAllLines.length > 1000) {
        logAllLines = logAllLines.slice(-800);
    }

    // If line matches current filter (or no filter), append to visible output
    if (!logFilterText || line.toLowerCase().includes(logFilterText)) {
        if ($logOutput.textContent.length > 0) {
            $logOutput.textContent += '\n' + line;
        } else {
            $logOutput.textContent = line;
        }

        if (logAutoFollow) {
            $logContainer.scrollTop = $logContainer.scrollHeight;
        }
    }
}

function renderFilteredLogs() {
    const filtered = logFilterText
        ? logAllLines.filter(l => l.toLowerCase().includes(logFilterText))
        : logAllLines;
    $logOutput.textContent = filtered.join('\n');
}

// ── Toolbar actions ────────────────────────────────────
function bindToolbar() {
    document.getElementById('btn-import-file').addEventListener('click', async () => {
        try {
            const imported = await configSvc().ImportFiles();
            if (imported && imported.length > 0) {
                await refreshTree();
            }
        } catch (e) {
            showError('Import failed: ' + e);
        }
    });

    document.getElementById('btn-import-folder').addEventListener('click', async () => {
        try {
            const imported = await configSvc().ImportFolder();
            if (imported && imported.length > 0) {
                await refreshTree();
            }
        } catch (e) {
            showError('Import failed: ' + e);
        }
    });

    document.getElementById('btn-new-tunnel').addEventListener('click', async () => {
        const name = await showPrompt('New Tunnel', 'Enter tunnel name:');
        if (!name) return;
        try {
            const template = await configSvc().GetDefaultTemplate();
            await configSvc().AddConfig(name, template);
            await refreshTree();
        } catch (e) {
            showError('Failed to create tunnel: ' + e);
        }
    });

    document.getElementById('btn-new-folder').addEventListener('click', async () => {
        const name = await showPrompt('New Folder', 'Enter folder name:');
        if (!name) return;
        try {
            await configSvc().CreateFolder(name);
            await refreshTree();
        } catch (e) {
            showError('Failed to create folder: ' + e);
        }
    });

    $btnDelete.addEventListener('click', async () => {
        if (!selectedType || !selectedId) return;

        const what = selectedType === 'folder'
            ? `folder "${selectedId}" and all its tunnels`
            : `tunnel "${selectedId.replace(/\.conf$/, '')}"`;

        const confirmed = await showConfirm('Delete', `Are you sure you want to delete ${what}?`);
        if (!confirmed) return;

        try {
            if (selectedType === 'folder') {
                await configSvc().DeleteFolder(selectedId);
            } else {
                await configSvc().DeleteTunnel(selectedId);
            }
            showEmptyView();
            await refreshTree();
        } catch (e) {
            showError('Delete failed: ' + e);
        }
    });

    $btnDisconnect.addEventListener('click', async () => {
        try {
            await tunnelSvc().Disconnect();
        } catch (e) {
            showError('Disconnect failed: ' + e);
        }
    });

    // Import to folder button
    document.getElementById('btn-import-to-folder').addEventListener('click', async () => {
        if (selectedType !== 'folder') return;
        try {
            const imported = await configSvc().ImportFilesToFolder(selectedId);
            if (imported && imported.length > 0) {
                await refreshTree();
                await showFolderView(selectedId);
            }
        } catch (e) {
            showError('Import failed: ' + e);
        }
    });
}

// ── Editor actions ─────────────────────────────────────
function bindEditorActions() {
    document.getElementById('btn-connect-tunnel').addEventListener('click', async () => {
        if (selectedType !== 'tunnel' || !selectedId) return;
        try {
            await tunnelSvc().Connect(selectedId);
        } catch (e) {
            showError('Connect failed: ' + e);
        }
    });

    document.getElementById('btn-connect-folder').addEventListener('click', async () => {
        if (selectedType !== 'folder' || !selectedId) return;
        try {
            await tunnelSvc().ConnectRandom(selectedId);
        } catch (e) {
            showError('Connect failed: ' + e);
        }
    });

    document.getElementById('btn-save').addEventListener('click', async () => {
        if (selectedType !== 'tunnel' || !selectedId) return;
        try {
            await configSvc().SaveConfig(selectedId, $editor.value);
            // Re-read to update info chips
            await showTunnelView(selectedId);
        } catch (e) {
            showError('Save failed: ' + e);
        }
    });

    document.getElementById('btn-rename').addEventListener('click', async () => {
        if (selectedType !== 'tunnel' || !selectedId) return;
        const currentName = selectedId.replace(/\.conf$/, '');
        const newName = await showPrompt('Rename Tunnel', currentName);
        if (!newName || newName === currentName) return;
        try {
            await configSvc().RenameTunnel(selectedId, newName);
            const newFilename = newName.endsWith('.conf') ? newName : newName + '.conf';
            selectedId = newFilename;
            await refreshTree();
            await showTunnelView(newFilename);
        } catch (e) {
            showError('Rename failed: ' + e);
        }
    });

    $folderSelect.addEventListener('change', async () => {
        const folder = $folderSelect.value;
        if (selectedType !== 'tunnel' || !selectedId) return;
        // Reset dropdown visual
        const filename = selectedId;
        $folderSelect.selectedIndex = 0;
        try {
            await configSvc().MoveToFolder(filename, folder);
            await refreshTree();
        } catch (e) {
            showError('Move failed: ' + e);
        }
    });
}

// ── Dialogs ────────────────────────────────────────────
function bindDialogs() {
    document.getElementById('error-dialog-close').addEventListener('click', () => {
        document.getElementById('error-dialog').close();
    });
}

function showError(message) {
    const dialog = document.getElementById('error-dialog');
    document.getElementById('error-dialog-message').textContent = String(message);
    dialog.showModal();
}

function showPrompt(title, placeholder) {
    return new Promise((resolve) => {
        const dialog = document.getElementById('prompt-dialog');
        const input = document.getElementById('prompt-dialog-input');
        document.getElementById('prompt-dialog-title').textContent = title;
        input.placeholder = placeholder || 'Name';
        input.value = '';

        const cleanup = () => {
            dialog.close();
            okBtn.removeEventListener('click', onOk);
            cancelBtn.removeEventListener('click', onCancel);
            input.removeEventListener('keydown', onKey);
        };

        const onOk = () => { cleanup(); resolve(input.value.trim()); };
        const onCancel = () => { cleanup(); resolve(null); };
        const onKey = (e) => { if (e.key === 'Enter') onOk(); if (e.key === 'Escape') onCancel(); };

        const okBtn = document.getElementById('prompt-dialog-ok');
        const cancelBtn = document.getElementById('prompt-dialog-cancel');
        okBtn.addEventListener('click', onOk);
        cancelBtn.addEventListener('click', onCancel);
        input.addEventListener('keydown', onKey);

        dialog.showModal();
        input.focus();
    });
}

function showConfirm(title, message) {
    return new Promise((resolve) => {
        const dialog = document.getElementById('confirm-dialog');
        document.getElementById('confirm-dialog-title').textContent = title;
        document.getElementById('confirm-dialog-message').textContent = message;

        const cleanup = () => {
            dialog.close();
            okBtn.removeEventListener('click', onOk);
            cancelBtn.removeEventListener('click', onCancel);
        };

        const onOk = () => { cleanup(); resolve(true); };
        const onCancel = () => { cleanup(); resolve(false); };

        const okBtn = document.getElementById('confirm-dialog-ok');
        const cancelBtn = document.getElementById('confirm-dialog-cancel');
        okBtn.addEventListener('click', onOk);
        cancelBtn.addEventListener('click', onCancel);

        dialog.showModal();
    });
}
