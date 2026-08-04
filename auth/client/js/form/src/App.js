const API_ERR = {
    ERR_NO_LOGIN_PWD: 1012,
    ERR_WRONG_LOGIN_PWD: 1013,
    ERR_INVAL_LOGIN: 1014,
    ERR_INVAL_PWD: 1015,
    ERR_LOGIN_EXISTS: 1016
};

const ERR = {
    NONE                   : 'none',
    FATAL                  : 'fatal',
    LOGIN_NEED_LOGIN       : '_login_need_login',
    LOGIN_NEED_PWD         : '_login_need_pwd',
    LOGIN_BAD_REQUEST      : '_login_bad_request',
    LOGIN_NOT_FOUND        : '_login_not_found',
    SIGNUP_NEED_LOGIN      : '_signup_need_login',
    SIGNUP_ILLEGAL_LOGIN   : '_signup_illegal_login',
    SIGNUP_LOGIN_EXISTS    : '_signup_login_exists',
    SIGNUP_NEED_PWD        : '_signup_need_pwd',
    SIGNUP_WEAK_PWD        : '_signup_weak_pwd',
    SIGNUP_NEED_PWD_CONFIRM: '_signup_need_pwd_confirm',
    SIGNUP_PWD_MISMATCH    : '_signup_pwd_mismatch',
    SIGNUP_BAD_REQUEST     : '_signup_bad_request',
};

class View {
    constructor() {
        this.loginMark = document.querySelector('._loginMark');
        this.signupMark = document.querySelector('._signupMark');
        this.loginFormBox = document.querySelector('._loginForm');
        this.signupFormBox = document.querySelector('._signupForm');
        this.okButton = document.querySelector('._okButton');

        this.loginInputs = {
            login: document.querySelector('._inp_login_login'),
            password: document.querySelector('._inp_login_pwd'),
        };
        this.signupInputs = {
            login: document.querySelector('._inp_signup_login'),
            password: document.querySelector('._inp_signup_pwd'),
            confirm: document.querySelector('._inp_signup_pwd_confirm'),
        };
    }

    hideWarnings() {
        for (let i in ERR) {
            if (i == 'NONE' || i == 'FATAL') continue;
            let warningBox = document.querySelector('.' + ERR[i]);
            warningBox.classList.add('auth-input-warning-hidden');
        }
    }

    /**
     * @param {string} cssClass 
     */
    showWarning(cssClass) {
        const warningBox = document.querySelector('.' + cssClass);
        warningBox.classList.remove('auth-input-warning-hidden');
    }

    showForm() {
        document.querySelector('._form').classList.remove('auth-mode-hidden');
        document.querySelector('._msg_success').classList.add('auth-mode-hidden');
        document.querySelector('._msg_fail').classList.add('auth-mode-hidden');
    }

    showSuccess() {
        document.querySelector('._form').classList.add('auth-mode-hidden');
        document.querySelector('._msg_success').classList.remove('auth-mode-hidden');
        document.querySelector('._msg_fail').classList.add('auth-mode-hidden');
    }

    showFail() {
        document.querySelector('._form').classList.add('auth-mode-hidden');
        document.querySelector('._msg_success').classList.add('auth-mode-hidden');
        document.querySelector('._msg_fail').classList.remove('auth-mode-hidden');
    }
}

class LoginForm {
    /**
     * @param {View} view 
     */
    constructor(view) {
        this.view = view;
    }

    /**
     * @returns {string}
     */
    checkReadyToSubmit() {
        const v = this.view;
        if (v.loginInputs.login.value == '')
            return ERR.LOGIN_NEED_LOGIN;
        if (v.loginInputs.password.value == '')
            return ERR.LOGIN_NEED_PWD;    
        return ERR.NONE;
    }

    /**
     * @returns {string}
     */
    async submit() {
        const v = this.view;
        try {
            const response = await fetch('/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    login: v.loginInputs.login.value,
                    password: v.loginInputs.password.value
                })
            });
    
            if (!response.ok) {
                if (response.status === 400)
                    return ERR.LOGIN_BAD_REQUEST;
                if (response.status === 401 || response.status === 404)
                    return ERR.LOGIN_NOT_FOUND;
                throw new Error(`Response failed: ${response.status}`);
            }
    
            const data = await response.json();
            if (data.success)
                return ERR.NONE;
    
            console.error(error);
            return ERR.FATAL;
        } catch (error) {
            console.error(error);
            return ERR.FATAL;
        }
    }
}

class SignupForm {
    /**
     * @param {View} view 
     */
    constructor(view) {
        this.view = view;
    }

    /**
     * @returns {string}
     */
    checkReadyToSubmit() {
        const v = this.view;                
        if (v.signupInputs.login.value == '')
            return ERR.SIGNUP_NEED_LOGIN;
        const val = v.signupInputs.login.value;
        if (!this._validateLogin(val))
            return ERR.SIGNUP_ILLEGAL_LOGIN;
        if (v.signupInputs.password.value == '')
            return ERR.SIGNUP_NEED_PWD;
        const pwd = v.signupInputs.password.value;
        if (!this._validatePassword(pwd))
            return ERR.SIGNUP_WEAK_PWD;
        if (v.signupInputs.confirm.value == '')
            return ERR.SIGNUP_NEED_PWD_CONFIRM;
        if (v.signupInputs.password.value != v.signupInputs.confirm.value)
            return ERR.SIGNUP_PWD_MISMATCH;
        return ERR.NONE;
    }

    /**
     * @returns {string}
     */
    async submit() {
        const v = this.view;
        try {
            const response = await fetch('/signup', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    login: v.signupInputs.login.value,
                    password: v.signupInputs.password.value
                })
            });
    
            if (!response.ok) {
                if (response.status == 409)
                    return ERR.SIGNUP_LOGIN_EXISTS;
                if (response.status != 400)
                    throw new Error(`Response failed: ${response.status}`);
            }

            const data = await response.json();
            if (data.success)
                return ERR.NONE;

            switch (data.error_code) {
                case API_ERR.ERR_NO_LOGIN_PWD: return ERR.SIGNUP_BAD_REQUEST;
                case API_ERR.ERR_INVAL_LOGIN: return ERR.SIGNUP_ILLEGAL_LOGIN;
                case API_ERR.ERR_INVAL_PWD: return ERR.SIGNUP_WEAK_PWD;
            }
            
            console.error(error);
            return ERR.FATAL;
        } catch (error) {
            console.error(error);
            return ERR.FATAL;
        }
    }

    /**
     * @param {string} login
     * @returns {bool}
     */
    _validateLogin(login) {
        if (login.length < 3 || login.length > 20)
            return false;
        const doubleRegex = /(\.\.|__)/;
        const loginRegex = /^[a-zA-Z0-9_.]+$/;
        return !doubleRegex.test(login) && loginRegex.test(login);
    }

    /**
     * @param {string} password
     * @returns {bool}
     */
    _validatePassword(password) {
        if (password.length < 8)
            return false;
        if (!/[a-z]/.test(password) || !/[A-Z]/.test(password) || !/\d/.test(password))
            return false;
        if (!/[!@#$%^&*(),.?":{}|<>]/.test(password))
            return false;
        return true;
    }
}

class Form {
    /**
     * @param {View} view 
     */
    constructor(view) {
        this.view = view;
        this.mode = 'login';
        this.loginForm = new LoginForm(view);
        this.signupForm = new SignupForm(view);
        this.activeForm = this.loginForm;
        this.locked = false;

        _initHandlers(this)
    }

    /**
     * @returns {bool}
     */
    isLocked() {
        return this.locked;
    }

    lock() {
        this.locked = true;
    }

    unlock() {
        this.locked = false;        
    }

    setModeLogin() {
        if (this.isLocked()) return;
        if (this.mode == 'login') return;
        const v = this.view;
        v.loginMark.classList.remove('auth-mark-inactive-dark');
        v.signupMark.classList.add('auth-mark-inactive-dark');
        v.signupFormBox.style.display = 'none';
        v.loginFormBox.style.display = null;
        _alignEye(v.loginInputs.password);
        this.mode = 'login';
        this.activeForm = this.loginForm;
    }

    setModeSignup() {
        if (this.isLocked()) return;
        if (this.mode == 'signup') return;
        const v = this.view;
        v.signupMark.classList.remove('auth-mark-inactive-dark');
        v.loginMark.classList.add('auth-mark-inactive-dark');
        v.loginFormBox.style.display = 'none';
        v.signupFormBox.style.display = null;
        _alignEye(v.signupInputs.password);
        _alignEye(v.signupInputs.confirm);
        this.mode = 'signup';
        this.activeForm = this.signupForm;
    }

    async submit() {
        if (this.isLocked()) return;

        let err = this.activeForm.checkReadyToSubmit();
        if (err != ERR.NONE) {
            this.view.showWarning(err);
            return;
        }

        err = await this.activeForm.submit();
        if (err == ERR.FATAL) {
            console.log(err);
            this.view.showFail();
            return;
        }

        if (err != ERR.NONE) {
            this.view.showWarning(err);
            return;
        }

        this.view.showSuccess();
        setTimeout(()=>{

            //TODO test
            window.location.href = '/return';

        }, 700);
    }
}

/**
 * @private
 * @param {Form} self 
 */
function _initHandlers(self) {
    const v = self.view;
    v.loginMark.addEventListener('mouseup', ()=>self.setModeLogin());
    v.signupMark.addEventListener('mouseup', ()=>self.setModeSignup());

    v.okButton.addEventListener('mouseup', ()=>self.submit());

    for (let i in v.loginInputs)
        v.loginInputs[i].addEventListener('input', ()=>_onInput(v));
    for (let i in v.signupInputs)
        v.signupInputs[i].addEventListener('input', ()=>_onInput(v));

    //TODO v.signupInputs.login - on 'input' check existing logins

    _makePwdEye(v.loginInputs.password);
    _makePwdEye(v.signupInputs.password);
    _makePwdEye(v.signupInputs.confirm);
}

function _makePwdEye(input) {
    _alignEye(input);
    const wrapper = input.parentNode.querySelector('.auth-eye-wrapper');
    const eye = wrapper.querySelector('.auth-eye');

    input._lxgo_isHidden = true;
    wrapper.addEventListener('mouseup', ()=>{
        input._lxgo_isHidden = !input._lxgo_isHidden;
        input._lxgo_isHidden
            ? _closeEye(input, eye)
            : _openEye(input, eye);
    });
}

function _openEye(input, eye) {
    input.removeAttribute('type');
    eye.classList.remove('auth-eye-closed');
    eye.classList.add('auth-eye-opened');
}

function _closeEye(input, eye) {
    input.setAttribute('type', 'password');
    eye.classList.remove('auth-eye-opened');
    eye.classList.add('auth-eye-closed');    
}

function _alignEye(input) {
    input.parentNode.querySelector('.auth-eye-wrapper').style.height = input.offsetHeight + 'px';
}

function _onInput(view) {
    view.hideWarnings();
}

document.addEventListener("DOMContentLoaded", function(e) {
    const view = new View();
    const form = new Form(view);
});
