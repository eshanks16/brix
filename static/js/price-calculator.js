/**
 * Real-time Pizza Price Calculator
 * Calculates and displays the total price as users build their pizza
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
        // Get form elements
        const sizeSelect = document.getElementById('size');
        const form = document.querySelector('.order-form form');

        if (!sizeSelect || !form) {
            return; // Not on order page
        }

        // Create price display element
        createPriceDisplay();

        // Add event listeners
        sizeSelect.addEventListener('change', updatePrice);

        // Listen to all topping radio buttons
        const toppingRadios = document.querySelectorAll('input[type="radio"][name^="topping_"]');
        toppingRadios.forEach(radio => {
            radio.addEventListener('change', updatePrice);
        });

        // Initial calculation
        updatePrice();
    }

    function createPriceDisplay() {
        const form = document.querySelector('.order-form form');
        const submitButton = form.querySelector('.btn-submit');

        // Create price summary container
        const priceContainer = document.createElement('div');
        priceContainer.className = 'price-summary';
        priceContainer.innerHTML = `
            <div class="price-breakdown">
                <div class="price-line">
                    <span class="price-label">Base Price:</span>
                    <span class="price-value" id="base-price">$0.00</span>
                </div>
                <div class="price-line">
                    <span class="price-label">Toppings (<span id="topping-count">0</span>):</span>
                    <span class="price-value" id="toppings-price">$0.00</span>
                </div>
                <div class="price-line price-total">
                    <span class="price-label">Total:</span>
                    <span class="price-value" id="total-price">$0.00</span>
                </div>
            </div>
        `;

        // Insert before submit button
        form.insertBefore(priceContainer, submitButton);
    }

    function updatePrice() {
        const sizeSelect = document.getElementById('size');
        const selectedOption = sizeSelect.options[sizeSelect.selectedIndex];

        // Get base price from selected size
        let basePrice = 0;
        if (selectedOption && selectedOption.value) {
            // Extract price from option text: "Medium (12") - $15.99"
            const priceMatch = selectedOption.text.match(/\$(\d+\.\d+)/);
            if (priceMatch) {
                basePrice = parseFloat(priceMatch[1]);
            }
        }

        // Calculate toppings price
        const toppingsPrice = calculateToppingsPrice();
        const toppingCount = countSelectedToppings();

        // Calculate total
        const total = basePrice + toppingsPrice;

        // Update display with animation
        animatePrice('base-price', basePrice);
        animatePrice('toppings-price', toppingsPrice);
        animatePrice('total-price', total);

        // Update topping count
        const toppingCountEl = document.getElementById('topping-count');
        if (toppingCountEl) {
            toppingCountEl.textContent = toppingCount;
        }
    }

    function calculateToppingsPrice() {
        let total = 0;

        // Get all checked radio buttons
        const checkedRadios = document.querySelectorAll('input[type="radio"][name^="topping_"]:checked');

        checkedRadios.forEach(radio => {
            // Find the topping name element by traversing up and finding the first .topping-name in the same row
            let currentElement = radio.closest('.topping-radio');

            // Go back through siblings to find the topping-name
            while (currentElement && currentElement.previousElementSibling) {
                currentElement = currentElement.previousElementSibling;
                if (currentElement.classList && currentElement.classList.contains('topping-name')) {
                    const priceMatch = currentElement.textContent.match(/\$(\d+\.\d+)/);
                    if (priceMatch) {
                        total += parseFloat(priceMatch[1]);
                    }
                    break;
                }
            }
        });

        return total;
    }

    function countSelectedToppings() {
        const selectedToppings = new Set();
        const checkedRadios = document.querySelectorAll('input[type="radio"][name^="topping_"]:checked');

        checkedRadios.forEach(radio => {
            const toppingName = radio.name.replace('topping_', '');
            selectedToppings.add(toppingName);
        });

        return selectedToppings.size;
    }

    function animatePrice(elementId, newPrice) {
        const element = document.getElementById(elementId);
        if (!element) return;

        const formattedPrice = '$' + newPrice.toFixed(2);

        // Add pulse animation class
        element.classList.remove('price-pulse');

        // Force reflow to restart animation
        void element.offsetWidth;

        // Update text and add animation
        element.textContent = formattedPrice;
        element.classList.add('price-pulse');
    }
})();
