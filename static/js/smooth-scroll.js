/**
 * Smooth Scroll Navigation
 * Provides smooth scrolling behavior for navigation links and anchor links
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
        setupSmoothScroll();
        handleInitialHash();
    }

    function setupSmoothScroll() {
        // Handle all anchor links with hash
        document.addEventListener('click', function(e) {
            const target = e.target.closest('a[href^="#"]');

            if (!target) return;

            const hash = target.getAttribute('href');
            if (hash === '#') return;

            // Check if target element exists on current page
            const targetElement = document.querySelector(hash);
            if (!targetElement) return;

            // Prevent default jump
            e.preventDefault();

            // Smooth scroll to element
            scrollToElement(targetElement, hash);
        });

        // Add smooth scroll to "back to top" functionality
        addBackToTop();
    }

    function scrollToElement(element, hash) {
        // Scroll with smooth behavior
        element.scrollIntoView({
            behavior: 'smooth',
            block: 'start'
        });

        // Update URL hash without jumping
        if (history.pushState) {
            history.pushState(null, null, hash);
        } else {
            // Fallback for older browsers
            window.location.hash = hash;
        }

        // Set focus for accessibility
        element.setAttribute('tabindex', '-1');
        element.focus();
    }

    function handleInitialHash() {
        // Handle hash in URL on page load
        if (window.location.hash) {
            const targetElement = document.querySelector(window.location.hash);
            if (targetElement) {
                // Small delay to ensure page is fully rendered
                setTimeout(function() {
                    scrollToElement(targetElement, window.location.hash);
                }, 100);
            }
        }
    }

    function addBackToTop() {
        // Check if we should show a "back to top" button
        let backToTopBtn = document.getElementById('back-to-top');

        // Create button if it doesn't exist
        if (!backToTopBtn) {
            backToTopBtn = document.createElement('button');
            backToTopBtn.id = 'back-to-top';
            backToTopBtn.className = 'back-to-top';
            backToTopBtn.innerHTML = '↑';
            backToTopBtn.title = 'Back to top';
            backToTopBtn.setAttribute('aria-label', 'Scroll to top');
            document.body.appendChild(backToTopBtn);
        }

        // Show/hide button based on scroll position
        window.addEventListener('scroll', function() {
            if (window.pageYOffset > 300) {
                backToTopBtn.classList.add('visible');
            } else {
                backToTopBtn.classList.remove('visible');
            }
        });

        // Scroll to top on click
        backToTopBtn.addEventListener('click', function() {
            window.scrollTo({
                top: 0,
                behavior: 'smooth'
            });
        });
    }

    // Add smooth scrolling support for browsers that don't support it natively
    function polyfillSmoothScroll() {
        // Check if smooth scroll is supported
        if ('scrollBehavior' in document.documentElement.style) {
            return;
        }

        // Simple polyfill for older browsers
        const originalScrollTo = window.scrollTo;
        window.scrollTo = function(options) {
            if (typeof options === 'object' && options.behavior === 'smooth') {
                const start = window.pageYOffset;
                const target = options.top || 0;
                const duration = 500;
                const startTime = performance.now();

                function scrollAnimation(currentTime) {
                    const elapsed = currentTime - startTime;
                    const progress = Math.min(elapsed / duration, 1);

                    // Easing function
                    const easing = progress * (2 - progress);

                    const position = start + (target - start) * easing;
                    window.scrollTo(0, position);

                    if (progress < 1) {
                        requestAnimationFrame(scrollAnimation);
                    }
                }

                requestAnimationFrame(scrollAnimation);
            } else {
                originalScrollTo.apply(window, arguments);
            }
        };
    }

    polyfillSmoothScroll();
})();
