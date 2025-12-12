/**
 * Registration Form Validation
 * Provides real-time validation for registration fields including phone number
 */

(function() {
    'use strict';

    // Wait for DOM to be ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

    function init() {
        const form = document.querySelector('.auth-form form');
        if (!form || form.action.indexOf('/register') === -1) return;

        setupValidation(form);
    }

    function setupValidation(form) {
        const firstNameInput = document.getElementById('first_name');
        const lastNameInput = document.getElementById('last_name');
        const emailInput = document.getElementById('email');
        const phoneInput = document.getElementById('phone');
        const passwordInput = document.getElementById('password');

        // Phone number validation
        if (phoneInput) {
            phoneInput.addEventListener('input', function() {
                validatePhone(this);
            });

            phoneInput.addEventListener('blur', function() {
                validatePhone(this);
            });
        }

        // Email validation
        if (emailInput) {
            emailInput.addEventListener('blur', function() {
                validateEmail(this);
            });
        }

        // Name validation
        if (firstNameInput) {
            firstNameInput.addEventListener('blur', function() {
                validateName(this, 'First name');
            });
        }

        if (lastNameInput) {
            lastNameInput.addEventListener('blur', function() {
                validateName(this, 'Last name');
            });
        }

        // Password validation
        if (passwordInput) {
            passwordInput.addEventListener('input', function() {
                validatePassword(this);
            });
        }

        // Form submission validation
        form.addEventListener('submit', function(e) {
            let isValid = true;

            if (firstNameInput && !validateName(firstNameInput, 'First name')) {
                isValid = false;
            }

            if (lastNameInput && !validateName(lastNameInput, 'Last name')) {
                isValid = false;
            }

            if (emailInput && !validateEmail(emailInput)) {
                isValid = false;
            }

            if (phoneInput && !validatePhone(phoneInput)) {
                isValid = false;
            }

            if (passwordInput && !validatePassword(passwordInput)) {
                isValid = false;
            }

            if (!isValid) {
                e.preventDefault();

                // Scroll to first error
                const firstError = form.querySelector('.field-error');
                if (firstError) {
                    firstError.scrollIntoView({ behavior: 'smooth', block: 'center' });
                }
            }
        });
    }

    function validatePhone(input) {
        const value = input.value.trim();

        if (!value) {
            showError(input, 'Phone number is required');
            return false;
        }

        // Remove all non-numeric characters to count digits
        const digitsOnly = value.replace(/\D/g, '');

        if (digitsOnly.length < 10) {
            showError(input, 'Phone number must be at least 10 digits');
            return false;
        }

        if (digitsOnly.length > 15) {
            showError(input, 'Phone number is too long');
            return false;
        }

        // Check for valid format (allows spaces, dashes, parentheses, plus)
        const phonePattern = /^[\d\s\-\(\)\+]+$/;
        if (!phonePattern.test(value)) {
            showError(input, 'Please use only numbers, spaces, dashes, parentheses, or +');
            return false;
        }

        showSuccess(input, 'Valid phone number');
        return true;
    }

    function validateEmail(input) {
        const value = input.value.trim();

        if (!value) {
            showError(input, 'Email address is required');
            return false;
        }

        // Simple email validation
        const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        if (!emailPattern.test(value)) {
            showError(input, 'Please enter a valid email address');
            return false;
        }

        showSuccess(input, 'Valid email');
        return true;
    }

    function validateName(input, fieldName) {
        const value = input.value.trim();

        if (!value) {
            showError(input, fieldName + ' is required');
            return false;
        }

        if (value.length < 2) {
            showError(input, fieldName + ' must be at least 2 characters');
            return false;
        }

        if (value.length > 50) {
            showError(input, fieldName + ' is too long');
            return false;
        }

        // Check for valid characters (letters, spaces, hyphens, apostrophes)
        const namePattern = /^[A-Za-z\s\-']+$/;
        if (!namePattern.test(value)) {
            showError(input, 'Please use only letters, spaces, hyphens, and apostrophes');
            return false;
        }

        showSuccess(input);
        return true;
    }

    function validatePassword(input) {
        const value = input.value;

        if (!value) {
            showError(input, 'Password is required');
            return false;
        }

        if (value.length < 6) {
            showError(input, 'Password must be at least 6 characters');
            return false;
        }

        if (value.length > 100) {
            showError(input, 'Password is too long');
            return false;
        }

        showSuccess(input, 'Strong password');
        return true;
    }

    function showError(input, message) {
        const formGroup = input.closest('.form-group');
        if (!formGroup) return;

        // Remove existing feedback
        clearFeedback(formGroup);

        // Add error class
        formGroup.classList.add('field-error');
        formGroup.classList.remove('field-valid');

        // Add error message
        const errorMsg = document.createElement('div');
        errorMsg.className = 'field-error-message';
        errorMsg.textContent = message;
        formGroup.appendChild(errorMsg);

        // Style the input
        input.style.borderColor = '#C0392B';
        input.style.boxShadow = '0 0 0 3px rgba(192, 57, 43, 0.1)';
    }

    function showSuccess(input, message) {
        const formGroup = input.closest('.form-group');
        if (!formGroup) return;

        // Remove existing feedback
        clearFeedback(formGroup);

        // Add valid class
        formGroup.classList.remove('field-error');
        formGroup.classList.add('field-valid');

        // Add success message if provided
        if (message) {
            const successMsg = document.createElement('div');
            successMsg.className = 'field-success-message';
            successMsg.textContent = message;
            formGroup.appendChild(successMsg);
        }

        // Style the input
        input.style.borderColor = '#3A5F3A';
        input.style.boxShadow = '0 0 0 3px rgba(58, 95, 58, 0.1)';
    }

    function clearFeedback(formGroup) {
        const existingError = formGroup.querySelector('.field-error-message');
        if (existingError) {
            existingError.remove();
        }

        const existingSuccess = formGroup.querySelector('.field-success-message');
        if (existingSuccess) {
            existingSuccess.remove();
        }
    }
})();
