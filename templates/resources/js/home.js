// Like toggle
function toggleLike(btn) {
    const countEl = btn.querySelector('span');
    const liked = btn.classList.toggle('liked');
    let count = parseInt(countEl.textContent);
    countEl.textContent = liked ? count + 1 : count - 1;
    btn.style.color = liked ? '#c8522a' : '';
}

// Comments toggle
function toggleComments(cardId) {
    const panel = document.getElementById('comments-' + cardId);
    panel.classList.toggle('open');
}

// Add comment
function addComment(cardId) {
    const input = document.getElementById('input-' + cardId);
    const text = input.value.trim();
    if (!text) return;

    const panel = document.getElementById('comments-' + cardId);
    const commentList = panel;

    const item = document.createElement('div');
    item.className = 'comment-item';
    item.innerHTML = `
        <div class="comment-avt"><img src="https://i.pravatar.cc/100?img=12" alt="You"/></div>
        <div class="comment-body">
        <div class="comment-user">You</div>
        <div class="comment-text">${text}</div>
        </div>
    `;
    // Insert before input row
    const inputRow = panel.querySelector('.comment-input-row');
    panel.insertBefore(item, inputRow);
    input.value = '';
}

// Follow toggle
function toggleFollow(btn) {
    const following = btn.classList.toggle('following');
    btn.textContent = following ? 'Following' : 'Follow';
}

// Tag filter (visual only)
function filterTag(el) {
    document.querySelectorAll('.tag-pill').forEach(t => t.classList.remove('active'));
    el.classList.add('active');
}
