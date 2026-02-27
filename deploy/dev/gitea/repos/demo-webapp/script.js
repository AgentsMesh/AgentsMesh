// Task list management
document.addEventListener("DOMContentLoaded", () => {
  const form = document.getElementById("task-form");
  const input = document.getElementById("task-input");
  const list = document.getElementById("task-list");
  const status = document.getElementById("status");

  let taskCount = list.children.length;

  form.addEventListener("submit", (e) => {
    e.preventDefault();
    const text = input.value.trim();
    if (!text) return;

    const li = document.createElement("li");
    li.textContent = text;
    li.addEventListener("click", () => {
      li.style.textDecoration =
        li.style.textDecoration === "line-through" ? "none" : "line-through";
      updateStatus();
    });

    list.appendChild(li);
    taskCount++;
    input.value = "";
    updateStatus();
  });

  // Click to toggle completion on existing items
  Array.from(list.children).forEach((li) => {
    li.addEventListener("click", () => {
      li.style.textDecoration =
        li.style.textDecoration === "line-through" ? "none" : "line-through";
      updateStatus();
    });
  });

  function updateStatus() {
    const total = list.children.length;
    const completed = Array.from(list.children).filter(
      (li) => li.style.textDecoration === "line-through"
    ).length;
    status.textContent = `${completed}/${total} completed`;
  }
});
