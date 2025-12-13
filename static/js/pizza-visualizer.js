/**
 * Interactive Pizza Visualizer
 * Shows a visual representation of the pizza as users select toppings
 */

(function() {
    'use strict';

    // Wait for DOM to be ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

    // Topping emoji mappings - using ingredient icons
    const toppingEmojis = {
        'Pepperoni': '🔴',
        'Mushrooms': '🍄',
        'Onions': '🧅',
        'Italian Sausage': '🟤',
        'Bacon': '🥓',
        'Extra Cheese': '🧀',
        'Black Olives': '⚫',
        'Bell Peppers': '🫑',
        'Pineapple': '🍍',
        'Spinach': '🥬',
        'Tomatoes': '🍅',
        'Jalapeños': '🌶️',
        'Ham': '🐷',
        'Chicken': '🐔',
        'Feta': '🟡'
    };

    function init() {
        const form = document.querySelector('.order-form form');
        if (!form) return;

        // Create pizza visualizer
        createPizzaVisualizer();

        // Listen to all topping radio buttons
        const toppingRadios = document.querySelectorAll('input[type="radio"][name^="topping_"]');
        toppingRadios.forEach(radio => {
            radio.addEventListener('change', updatePizzaVisualization);
        });

        // Initial visualization
        updatePizzaVisualization();
    }

    function createPizzaVisualizer() {
        const toppingsSection = document.querySelector('.toppings-section');

        const visualizer = document.createElement('div');
        visualizer.className = 'pizza-visualizer';
        visualizer.innerHTML = `
            <h3>Your Pizza Preview</h3>
            <div class="pizza-container">
                <svg class="pizza-svg" viewBox="0 0 300 300" xmlns="http://www.w3.org/2000/svg">
                    <!-- Pizza base -->
                    <circle cx="150" cy="150" r="140" fill="#F4E4C1" stroke="#D4A574" stroke-width="3"/>

                    <!-- Sauce layer -->
                    <circle cx="150" cy="150" r="135" fill="#D32F2F" opacity="0.7"/>

                    <!-- Cheese layer -->
                    <circle cx="150" cy="150" r="135" fill="#FFD54F" opacity="0.6"/>

                    <!-- Center line to show halves -->
                    <line x1="150" y1="10" x2="150" y2="290" stroke="#8B4513" stroke-width="2" opacity="0.3" stroke-dasharray="5,5"/>

                    <!-- Left half label -->
                    <text x="75" y="155" font-size="14" fill="#666" opacity="0.5" text-anchor="middle">L</text>

                    <!-- Right half label -->
                    <text x="225" y="155" font-size="14" fill="#666" opacity="0.5" text-anchor="middle">R</text>

                    <!-- Topping container groups -->
                    <g id="toppings-left"></g>
                    <g id="toppings-whole"></g>
                    <g id="toppings-right"></g>
                </svg>
            </div>
            <div class="topping-legend" id="topping-legend"></div>
        `;

        // Insert after the toppings grid
        const toppingsGrid = toppingsSection.querySelector('.toppings-grid');
        toppingsGrid.insertAdjacentElement('afterend', visualizer);
    }

    function updatePizzaVisualization() {
        const toppingsLeft = document.getElementById('toppings-left');
        const toppingsWhole = document.getElementById('toppings-whole');
        const toppingsRight = document.getElementById('toppings-right');
        const legend = document.getElementById('topping-legend');

        if (!toppingsLeft || !toppingsWhole || !toppingsRight || !legend) return;

        // Clear existing toppings
        toppingsLeft.innerHTML = '';
        toppingsWhole.innerHTML = '';
        toppingsRight.innerHTML = '';
        legend.innerHTML = '';

        const selectedToppings = [];

        // Get all checked radio buttons
        const checkedRadios = document.querySelectorAll('input[type="radio"][name^="topping_"]:checked');

        checkedRadios.forEach(radio => {
            if (radio.value === 'none') return;

            const toppingName = radio.name.replace('topping_', '');
            const placement = radio.value; // 'left', 'whole', or 'right'

            selectedToppings.push({ name: toppingName, placement });
        });

        // Add toppings to visualization
        selectedToppings.forEach((topping, index) => {
            const emoji = toppingEmojis[topping.name] || '🍕';

            // Generate random positions for this topping
            const positions = generateToppingPositions(topping.placement, 4 + index);

            positions.forEach(pos => {
                const text = document.createElementNS('http://www.w3.org/2000/svg', 'text');
                text.setAttribute('x', pos.x);
                text.setAttribute('y', pos.y);
                text.setAttribute('font-size', '20');
                text.setAttribute('text-anchor', 'middle');
                text.textContent = emoji;

                // Add to appropriate group
                if (topping.placement === 'left') {
                    toppingsLeft.appendChild(text);
                } else if (topping.placement === 'right') {
                    toppingsRight.appendChild(text);
                } else {
                    toppingsWhole.appendChild(text);
                }
            });

            // Add to legend
            const legendItem = document.createElement('span');
            legendItem.className = 'legend-item';
            legendItem.innerHTML = `${emoji} ${topping.name} <span class="placement">(${topping.placement})</span>`;
            legend.appendChild(legendItem);
        });

        if (selectedToppings.length === 0) {
            legend.innerHTML = '<span class="legend-empty">No toppings selected - plain cheese pizza</span>';
        }
    }

    function generateToppingPositions(placement, seed) {
        const positions = [];
        const count = 6; // Number of topping pieces to show
        const centerX = 150;
        const centerY = 150;
        const radius = 100; // How far from center to place toppings

        // Use seed for consistent random positions per topping
        const random = seededRandom(seed);

        for (let i = 0; i < count; i++) {
            let angle, x, y;

            if (placement === 'left') {
                // Left half: angles from 90° to 270°
                angle = (Math.PI / 2) + (random() * Math.PI);
            } else if (placement === 'right') {
                // Right half: angles from 270° to 90° (wrapping around)
                angle = (Math.PI * 3 / 2) + (random() * Math.PI);
            } else {
                // Whole pizza: distribute evenly around the circle with some randomness
                const baseAngle = (i / count) * Math.PI * 2;
                const randomOffset = (random() - 0.5) * (Math.PI / 3); // +/- 30 degrees
                angle = baseAngle + randomOffset;
            }

            // Random distance from center
            const distance = 30 + (random() * radius);

            x = centerX + Math.cos(angle) * distance;
            y = centerY + Math.sin(angle) * distance;

            // Make sure it's within the pizza
            const distFromCenter = Math.sqrt(Math.pow(x - centerX, 2) + Math.pow(y - centerY, 2));
            if (distFromCenter < 125) {
                positions.push({ x, y });
            }
        }

        return positions;
    }

    // Simple seeded random number generator for consistent positions
    function seededRandom(seed) {
        let value = seed;
        return function() {
            value = (value * 9301 + 49297) % 233280;
            return value / 233280;
        };
    }
})();
