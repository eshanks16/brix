/**
 * Form Validation with Enhanced UX
 * Provides real-time validation feedback and prevents common mistakes
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
        const form = document.querySelector('.order-form form');
        if (!form) return;

        // Add validation listeners
        setupValidation(form);
    }

    function setupValidation(form) {
        const pizzaStyleSelect = document.getElementById('pizza_style');
        const sizeSelect = document.getElementById('size');
        const submitButton = form.querySelector('.btn-submit');

        // Validate on form submission
        form.addEventListener('submit', function(e) {
            let isValid = true;
            let errors = [];

            // Check pizza style
            if (!pizzaStyleSelect.value) {
                isValid = false;
                errors.push('Please select a pizza style');
                markFieldAsInvalid(pizzaStyleSelect, 'Please select a pizza style');
            } else {
                markFieldAsValid(pizzaStyleSelect);
            }

            // Check size
            if (!sizeSelect.value) {
                isValid = false;
                errors.push('Please select a pizza size');
                markFieldAsInvalid(sizeSelect, 'Please select a pizza size');
            } else {
                markFieldAsValid(sizeSelect);
            }

            if (!isValid) {
                e.preventDefault();

                // Scroll to first error
                const firstError = form.querySelector('.field-error');
                if (firstError) {
                    firstError.scrollIntoView({ behavior: 'smooth', block: 'center' });
                }

                // Show summary error message
                showErrorSummary(errors);
            }
        });

        // Real-time validation on change
        pizzaStyleSelect.addEventListener('change', function() {
            if (this.value) {
                markFieldAsValid(this);
            }
        });

        sizeSelect.addEventListener('change', function() {
            if (this.value) {
                markFieldAsValid(this);
            }
        });

        // Add visual feedback for required fields
        addRequiredIndicators(form);
    }

    function markFieldAsInvalid(field, message) {
        const formGroup = field.closest('.form-group');
        if (!formGroup) return;

        // Remove existing error
        const existingError = formGroup.querySelector('.field-error-message');
        if (existingError) {
            existingError.remove();
        }

        // Add error class
        formGroup.classList.add('field-error');
        formGroup.classList.remove('field-valid');

        // Add error message
        const errorMsg = document.createElement('div');
        errorMsg.className = 'field-error-message';
        errorMsg.textContent = message;
        formGroup.appendChild(errorMsg);

        // Add red border to field
        field.style.borderColor = '#C0392B';
        field.style.boxShadow = '0 0 0 3px rgba(192, 57, 43, 0.1)';
    }

    function markFieldAsValid(field) {
        const formGroup = field.closest('.form-group');
        if (!formGroup) return;

        // Remove error
        const existingError = formGroup.querySelector('.field-error-message');
        if (existingError) {
            existingError.remove();
        }

        // Remove error class
        formGroup.classList.remove('field-error');

        // Add valid class if field has value
        if (field.value) {
            formGroup.classList.add('field-valid');
            field.style.borderColor = '#3A5F3A';
            field.style.boxShadow = '0 0 0 3px rgba(58, 95, 58, 0.1)';
        } else {
            formGroup.classList.remove('field-valid');
            field.style.borderColor = '';
            field.style.boxShadow = '';
        }
    }

    function showErrorSummary(errors) {
        // Remove existing summary
        const existingSummary = document.querySelector('.form-error-summary');
        if (existingSummary) {
            existingSummary.remove();
        }

        // Create error summary
        const summary = document.createElement('div');
        summary.className = 'form-error-summary';
        summary.innerHTML = `
            <div class="error-summary-header">
                <span class="error-icon">⚠️</span>
                <strong>Please fix the following errors:</strong>
            </div>
            <ul class="error-list">
                ${errors.map(error => `<li>${error}</li>`).join('')}
            </ul>
        `;

        // Insert at top of form
        const form = document.querySelector('.order-form form');
        form.insertBefore(summary, form.firstChild);

        // Auto-hide after 5 seconds
        setTimeout(() => {
            summary.style.opacity = '0';
            summary.style.transition = 'opacity 0.5s ease';
            setTimeout(() => summary.remove(), 500);
        }, 5000);
    }

    function addRequiredIndicators(form) {
        const pizzaStyleGroup = document.getElementById('pizza_style')?.closest('.form-group');
        const sizeGroup = document.getElementById('size')?.closest('.form-group');

        [pizzaStyleGroup, sizeGroup].forEach(group => {
            if (group) {
                const label = group.querySelector('label');
                if (label && !label.querySelector('.required-indicator')) {
                    const indicator = document.createElement('span');
                    indicator.className = 'required-indicator';
                    indicator.textContent = ' *';
                    indicator.title = 'Required field';
                    label.appendChild(indicator);
                }
            }
        });
    }
})();
