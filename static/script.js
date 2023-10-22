document.addEventListener("DOMContentLoaded", () => {
  const dropArea = document.getElementById("drop-area");
  const fileInputField = document.getElementById("file-input");

  document.getElementById("browse").addEventListener("click", (e) => {
    fileInputField.click();
  });

  fileInputField.addEventListener("change", (e) => {
    handleFileUpload(fileInputField.files[0]);
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

    const file = e.dataTransfer.files[0];
    handleFileUpload(file);
  });

  document.addEventListener("paste", (event) => {
    const clipboardData = event.clipboardData || window.clipboardData;

    // handle image
    const imageItem = Array.from(clipboardData.items).find((item) =>
      item.type.includes("image"),
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

  function handleResponse(response) {
    if (response.redirected) {
      window.location.href = response.url;
    }
  }

  async function fetchRequest(path, options) {
    try {
      const response = await fetch(path, options);
      handleResponse(response);
    } catch (error) {
      console.error("Error:", error);
    }
  }

  function handleFileUpload(file) {
    const formData = new FormData();
    formData.append("file", file);
    fetchRequest("/upload", {
      method: "POST",
      body: formData,
    });
  }

  async function handleURLUpload(url) {
    fetchRequest("/url", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ url: url }),
    });
  }
});
