document.addEventListener("DOMContentLoaded", () => {
  const dropArea = document.getElementById("drop-area");
  const fileInputField = document.getElementById("file-input");
  const centerText = document.getElementById("center-text");
  const spinner = document.getElementById("spinner");

  function ready() {
    centerText.style.display = "block";
    spinner.style.display = "none";
  }

  function busy() {
    centerText.style.display = "none";
    spinner.style.display = "block";
  }

  document.getElementById("cesspit").addEventListener("click", (e) => {
    fileInputField.click();
  });

  document.getElementById("browse").addEventListener("click", (e) => {
    fileInputField.click();
  });

  fileInputField.addEventListener("change", (e) => {
    if (fileInputField.files.length > 1) {
      handleMultipleFileUpload(fileInputField.files);
    } else if (fileInputField.files.length === 1) {
      handleFileUpload(fileInputField.files[0]);
    }
  });

  dropArea.addEventListener("dragover", (e) => {
    e.preventDefault();
    dropArea.classList.add("drag-over");
  });

  dropArea.addEventListener("dragleave", () => {
    dropArea.classList.remove("drag-over");
  });

  dropArea.addEventListener("drop", (e) => {
    e.preventDefault();
    dropArea.classList.remove("drag-over");

    if (e.dataTransfer.files.length > 1) {
      handleMultipleFileUpload(e.dataTransfer.files);
    } else if (e.dataTransfer.files.length === 1) {
      handleFileUpload(e.dataTransfer.files[0]);
    }
  });

  document.addEventListener("paste", (event) => {
    const clipboardData = event.clipboardData || window.clipboardData;

    // handle image
    const imageItem = Array.from(clipboardData.items).find((item) =>
      item.type.includes("image")
    );
    if (imageItem) {
      handleFileUpload(imageItem.getAsFile());
    }

    // handle url
    const text = clipboardData.getData("text");
    if (text.startsWith("http://") || text.startsWith("https://")) {
      handleURLUpload(text);
    }
  });

  function handleError(errorText) {
    const error = document.getElementById("error");
    error.textContent = `⚠ ${errorText}`;
    error.style.visibility = "visible";

    console.error("something goofed:", errorText);
  }

  async function handleResponse(response) {
    if (response.ok) {
      const result = await response.json();
      window.location.href = result.url;
    } else {
      const errorText = await response.text();
      handleError(errorText.toLowerCase());
    }
  }

  async function fetchRequest(path, options) {
    busy();
    try {
      const response = await fetch(path, options);
      handleResponse(response);
    } catch (error) {
      console.error("Error:", error);
    }
    ready();
  }

  function handleFileUpload(file) {
    const formData = new FormData();
    formData.append("file", file);
    fetchRequest("/upload", {
      method: "POST",
      headers: {
        Accept: "application/json",
      },
      body: formData,
    });
  }

  async function handleURLUpload(url) {
    fetchRequest("/url", {
      method: "POST",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ url: url }),
    });
  }

  function handleMultipleFileUpload(files) {
    const formData = new FormData();
    for (let i = 0; i < files.length; i++) {
      formData.append("file", files[i]);
    }
    fetchRequest("/upload", {
      method: "POST",
      headers: {
        Accept: "application/json",
      },
      body: formData,
    });
  }
});
