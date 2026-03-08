function switchTab(tab) {
    document.querySelectorAll('.auth-tab').forEach((t, i) => {
        t.classList.toggle('active', (i === 0 && tab === 'signin') || (i === 1 && tab === 'register'));
    });
    document.getElementById('form-signin').classList.toggle('active', tab === 'signin');
    document.getElementById('form-register').classList.toggle('active', tab === 'register');
}

function togglePw(id, btn) {
    const input = document.getElementById(id);
    const show = input.type === 'password';
    input.type = show ? 'text' : 'password';
    btn.textContent = show ? 'Hide' : 'Show';
}

function showError(id, show) {
    const el = document.getElementById(id);
    el.classList.toggle('show', show);
    const input = el.previousElementSibling?.tagName === 'DIV'
        ? el.previousElementSibling.querySelector('input')
        : el.previousElementSibling;
    if (input) input.classList.toggle('error', show);
}

function handleSignin() {
    let valid = true;
    const email = document.getElementById('signin-email').value;
    const pw = document.getElementById('signin-password').value;

    const emailOk = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
    showError('err-signin-email', !emailOk); if (!emailOk) valid = false;
    showError('err-signin-password', !pw); if (!pw) valid = false;

    if (valid) {
        document.getElementById('signin-success').classList.add('show');
        setTimeout(() => { window.location.href = 'home.html'; }, 1200);
    }
}

function handleRegister() {
    let valid = true;
    const username = document.getElementById('reg-username').value;
    const email = document.getElementById('reg-email').value;
    const pw = document.getElementById('reg-password').value;
    const confirm = document.getElementById('reg-confirm').value;

    const usernameOk = username.length >= 3 && username.length <= 50;
    showError('err-reg-username', !usernameOk); if (!usernameOk) valid = false;

    const emailOk = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
    showError('err-reg-email', !emailOk); if (!emailOk) valid = false;

    const pwOk = pw.length >= 8;
    showError('err-reg-password', !pwOk); if (!pwOk) valid = false;

    const confirmOk = pw === confirm && confirm.length > 0;
    showError('err-reg-confirm', !confirmOk); if (!confirmOk) valid = false;

    if (valid) {
        document.getElementById('register-success').classList.add('show');
        setTimeout(() => { window.location.href = 'home.html'; }, 1200);
    }
}
