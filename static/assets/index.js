const tokenName = "sx_access_token";
function getAccessToken() {
  return localStorage.getItem("sx_access_token") || "";
}

function setAccessToken(token) {
  localStorage.setItem("sx_access_token", token);
}

async function isAuthed() {
  const token = getAccessToken();
  if (!token) return false;

  const res = await fetch("/test-auth", {
    headers: {
      "X-SX-API-KEY": token,
    },
  });

  return res.status === 200;
}

async function doExport() {
  const token = getAccessToken();
  if (!token) return false;

  const res = await fetch("/export", {
    headers: {
      "X-SX-API-KEY": token,
    },
  });
  if (res.status !== 200) return false;

  const blob = await res.blob();

  const url = URL.createObjectURL(blob);

  const a = document.createElement("a");
  a.href = url;
  a.download = "export.zip";
  a.click();

  URL.revokeObjectURL(url);

  return true;
}

async function getFiles() {
  const token = getAccessToken();
  if (!token) return false;

  const res = await fetch("/files", {
    headers: {
      "X-SX-API-KEY": token,
    },
  });
  if (res.status !== 200) return false;

  const json = await res.json();
  if (!json.data) return false;

  return json.data;
}

async function doRenameFile(url, currentName) {
  const token = getAccessToken();
  if (!token) return false;

  let newName = prompt("What should this file be called?", currentName);
  if (!newName || newName.trim() == "") {
    return;
  }

  const res = await fetch(url, {
    method: "PUT",
    headers: {
      "X-SX-API-KEY": token,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      name: newName,
    }),
  });
  if (res.status !== 200) {
    alert("Failed to rename. Check console.");
    console.log(res);
    return;
  }

  window.location = window.location;
}

function buildFileEntriesHTML(files) {
  let html = "";

  for (let i = 0; i < files.length; i++) {
    const file = files[i];
    const url = "/f/" + file.id + file.ext;
    const deleteUrl = "/del?f=" + file.id + "&t=" + file.delete_token;

    switch (file.ext) {
      case ".png":
      case ".jpg":
        html += `
        <div class="file">
          <a href="${url}" class="img-container">
            <img src="${url}" alt="Preview Photo" />
          </a>
          <div class="content">
            <p class="title">${file.original_filename} (${file.id})</p>
            <div class="span">
              <a class="rename" onClick="doRenameFile('${url}', '${file.original_filename}')">Rename</a>
              <a href="${deleteUrl}">Delete</a>
            </div>
          </div>
        </div>
        `;
        break;
      default:
        html += `
        <div class="file">
          <a href="${url}" class="img-container">
            <p class="wrn-text">No preview available for this filetype. Click to open the file in a new tab.</p>
          </a>
          <div class="content">
            <p class="title">${file.original_filename} (${file.id})</p>
            <div class="span">
              <a class="rename" onClick="doRenameFile('${url}', '${file.original_filename}')">Rename</a>
              <a href="${deleteUrl}">Delete</a>
            </div>
          </div>
        </div>
        `;
        break;
    }
  }

  return html;
}

window.onload = async function () {
  // Token Management
  const tokenInput = document.getElementById("token_input");
  const saveButton = document.getElementById("save_button");

  tokenInput.value = getAccessToken();

  saveButton.addEventListener("click", () => {
    setAccessToken(tokenInput.value);
    window.location = window.location;
  });

  // Authentication Text
  const authed = await isAuthed();
  const topTextComponent = document.getElementById("top-text");
  topTextComponent.innerHTML = authed
    ? "You are authenticated."
    : "You are not authenticated.";
  topTextComponent.classList.add(authed ? "green" : "red");

  // Export
  const button = document.getElementById("export-btn");
  button.addEventListener("click", async () => {
    const result = await doExport();
    if (!result) return alert("Failed to export!");
  });

  // File Explorer
  const explorerContent = document.getElementById("file-explorer-grid");
  explorerContent.innerHTML = buildFileEntriesHTML(await getFiles());
};
