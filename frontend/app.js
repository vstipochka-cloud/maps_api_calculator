const app = {
    // Configuration - Backend API URL (dynamic for production support)
    get apiBaseURL() {
        // For localhost/development - use port 8080
        if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') {
            return 'http://localhost:8080';
        }
        // For production - use same host as frontend (Render, etc)
        return window.location.origin;
    },
    currentResults: null,
    apiTypes: [],
    apiDescriptions: {},
    providers: [],

    // Initialize app
    init() {
        this.setupNavigation();
        this.loadProvidersAndInit();
        this.setupEventListeners();
        this.populateAPITypesSection();
        console.log('API Calculator Frontend loaded');
        console.log('Backend API: ' + this.apiBaseURL);
    },

    // Load providers from server
    loadProvidersAndInit() {
        fetch(`${this.apiBaseURL}/providers`)
            .then(response => {
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                return response.json();
            })
            .then(data => {
                this.providers = data.providers || [];
                this.apiTypes = data.api_types || [];
                this.apiDescriptions = data.api_descriptions || {};
                this.populateAPISelectors();
                this.populateProviders();
                console.log('Loaded ' + this.providers.length + ' providers');
                console.log('Loaded ' + this.apiTypes.length + ' API types');
                console.log('Loaded ' + Object.keys(this.apiDescriptions).length + ' API descriptions');
            })
            .catch(error => {
                console.error('Failed to load providers:', error);
                alert('Failed to load providers from server. Make sure backend is running on ' + this.apiBaseURL);
            });
    },

    // Populate provider cards from server data
    populateProviders() {
        const providersGrid = document.querySelector('.providers-grid');
        if (!providersGrid) return;
        
        providersGrid.innerHTML = '';
        
        this.providers.forEach((provider, index) => {
            const card = document.createElement('div');
            card.className = 'provider-card';
            if (index === 0) {
                card.classList.add('best');
                card.innerHTML += '<span class="badge">Recommended</span>';
            }
            
            // Create a nicely formatted list of APIs
            const apisHtml = provider.apis.map(apiType => {
                const description = this.apiDescriptions[apiType];
                const displayName = description ? description.name : this.formatAPIName(apiType);
                return `
                    <li class="api-item">
                        <span class="api-name">${displayName}</span>
                    </li>
                `;
            }).join('');
            
            card.innerHTML += `
                <h3>${provider.name}</h3>
                <p class="desc">Supported APIs</p>
                <ul class="api-list">
                    ${apisHtml}
                </ul>
                <a href="${provider.url}" target="_blank" class="view-pricing-link">View Pricing →</a>
            `;
            
            providersGrid.appendChild(card);
        });
    },

    // Setup navigation
    setupNavigation() {
        document.querySelectorAll('.nav-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const page = e.target.dataset.page;
                this.goToPage(page);
            });
        });
    },

    // Navigate between pages
    goToPage(pageName) {
        // Hide all pages
        document.querySelectorAll('.page').forEach(page => {
            page.classList.remove('active');
        });

        // Show selected page
        document.getElementById(pageName).classList.add('active');

        // Update active nav button
        document.querySelectorAll('.nav-btn').forEach(btn => {
            btn.classList.remove('active');
            if (btn.dataset.page === pageName) {
                btn.classList.add('active');
            }
        });

        // Scroll to top
        window.scrollTo(0, 0);
    },

    // Populate API selectors
    populateAPISelectors() {
        const container = document.getElementById('apiSelectors');
        
        this.apiTypes.forEach(apiType => {
            const selector = document.createElement('div');
            selector.className = 'api-selector';
            
            const description = this.apiDescriptions[apiType];
            const displayName = description ? description.name : this.formatAPIName(apiType);
            const tooltipText = description ? description.description : '';
            
            const checkbox = document.createElement('input');
            checkbox.type = 'checkbox';
            checkbox.value = apiType;
            checkbox.id = `api-${apiType}`;
            
            const labelContainer = document.createElement('div');
            labelContainer.className = 'api-label-container';

            const label = document.createElement('label');
            label.textContent = displayName;
            label.htmlFor = `api-${apiType}`;
            label.className = 'api-selector-label';

            const infoToggle = document.createElement('button');
            infoToggle.type = 'button';
            infoToggle.className = 'api-info-toggle';
            infoToggle.textContent = 'ⓘ';
            infoToggle.title = 'Show description';

            labelContainer.appendChild(label);
            if (tooltipText) {
                labelContainer.appendChild(infoToggle);
            }
            
            const input = document.createElement('input');
            input.type = 'number';
            input.min = '0';
            input.max = '10000000';
            input.value = '0';
            input.placeholder = 'Enter requests';
            input.id = `count-${apiType}`;
            input.disabled = true;
            
            const inputContainer = document.createElement('div');
            inputContainer.className = 'api-input-container';
            inputContainer.appendChild(input);
            const unitSpan = document.createElement('span');
            unitSpan.className = 'api-unit';
            unitSpan.textContent = 'requests/month';
            inputContainer.appendChild(unitSpan);
            
            // Enable/disable input based on checkbox
            checkbox.addEventListener('change', (e) => {
                input.disabled = !e.target.checked;
                if (e.target.checked) {
                    input.focus();
                } else {
                    input.value = '0';
                }
            });
            
            const descSpan = document.createElement('div');
            descSpan.className = 'api-description-inline';
            descSpan.textContent = tooltipText;

            // Toggle description visibility (inline, small, non-layout-shifting)
            if (tooltipText) {
                infoToggle.addEventListener('click', (e) => {
                    e.preventDefault();
                    descSpan.classList.toggle('visible');
                    infoToggle.classList.toggle('active');
                });
            }
            
            selector.appendChild(checkbox);
            // place description inside label container so it doesn't create big gaps
            if (tooltipText) {
                labelContainer.appendChild(descSpan);
            }
            selector.appendChild(labelContainer);
            selector.appendChild(inputContainer);
            
            container.appendChild(selector);
        });
    },

    // Populate API types section with interactive cards
    populateAPITypesSection() {
        const container = document.querySelector('.api-grid');
        if (!container) return;
        
        container.innerHTML = '';
        
        this.apiTypes.forEach(apiType => {
            const description = this.apiDescriptions[apiType];
            const displayName = description ? description.name : this.formatAPIName(apiType);
            
            const tag = document.createElement('div');
            tag.className = 'api-tag-interactive';
            tag.textContent = displayName;
            
            container.appendChild(tag);
        });
    },

    // Format API names for display
    formatAPIName(apiType) {
        const description = this.apiDescriptions[apiType];
        if (description) {
            return description.name;
        }
        
        const names = {
            'geocoding': 'Geocoding',
            'routing': 'Routing',
            'map_tiles_raster': 'Map Tiles Raster',
            'map_tiles_vector_2d': 'Map Tiles Vector 2D',
            'map_tiles_vector_3d': 'Map Tiles Vector 3D',
            'static_maps': 'Static Maps',
            'static_street_view': 'Street View',
            'distance_matrix': 'Distance Matrix',
            'elevation': 'Elevation',
            'aerial_view': 'Aerial View'
        };
        return names[apiType] || apiType;
    },

    // Setup event listeners
    setupEventListeners() {
        // Handle Enter key in input
        document.addEventListener('keypress', (e) => {
            if (e.key === 'Enter' && e.target.type === 'number') {
                this.calculateCosts();
            }
        });
        
        // Update total elements when matrix params change
        const originsInput = document.getElementById('originsCount');
        const destinationsInput = document.getElementById('destinationsCount');
        
        if (originsInput && destinationsInput) {
            const updateTotalElements = () => this.updateTotalElements();
            originsInput.addEventListener('change', updateTotalElements);
            destinationsInput.addEventListener('change', updateTotalElements);
            originsInput.addEventListener('input', updateTotalElements);
            destinationsInput.addEventListener('input', updateTotalElements);
        }
        
        // Toggle visibility of matrix params section to save space
        const matrixToggle = document.getElementById('matrixToggleBtn');
        const matrixSection = document.getElementById('matrixParamsSection');
        if (matrixToggle && matrixSection) {
            matrixToggle.addEventListener('click', (e) => {
                e.preventDefault();
                const isHidden = matrixSection.classList.toggle('hidden');
                matrixToggle.setAttribute('aria-expanded', String(!isHidden));
                matrixToggle.textContent = isHidden ? 'Show' : 'Hide';
                if (!isHidden) {
                    // When revealed, ensure totals are up to date
                    this.updateTotalElements();
                }
            });
        }
    },
    
    // Update total elements display
    updateTotalElements() {
        const distanceMatrixCount = document.getElementById('count-distance_matrix');
        const originsInput = document.getElementById('originsCount');
        const destinationsInput = document.getElementById('destinationsCount');
        
        if (!distanceMatrixCount || !originsInput || !destinationsInput) return;
        
        const matrixRequests = parseInt(distanceMatrixCount.value) || 0;
        const origins = parseInt(originsInput.value) || 0;
        const destinations = parseInt(destinationsInput.value) || 0;
        
        const totalElements = matrixRequests * origins * destinations;
        document.getElementById('totalElements').textContent = totalElements.toLocaleString();
    },

    // Calculate costs
    calculateCosts() {
        // Collect selected APIs and their counts
        const apiRequests = {};
        let hasSelection = false;

        this.apiTypes.forEach(apiType => {
            const checkbox = document.getElementById(`api-${apiType}`);
            const input = document.getElementById(`count-${apiType}`);
            
            if (checkbox.checked) {
                const count = parseInt(input.value) || 0;
                if (count > 0) {
                    apiRequests[apiType] = count;
                    hasSelection = true;
                }
            }
        });

        if (!hasSelection) {
            alert('Please select at least one API and enter a number of requests');
            return;
        }

        // Get options
        const disableNewCustomerCredit = document.getElementById('disableNewCustomerCredit').checked;
        const disableFreeTier = document.getElementById('disableFreeTier').checked;
        const currency = document.getElementById('currencySelector').value;
        
        // Get matrix parameters if distance_matrix is selected
        let matrixParams = null;
        if (apiRequests.hasOwnProperty('distance_matrix')) {
            const originsCount = parseInt(document.getElementById('originsCount').value) || 0;
            const destinationsCount = parseInt(document.getElementById('destinationsCount').value) || 0;
            
            if (originsCount > 0 && destinationsCount > 0) {
                matrixParams = {
                    origins_count: originsCount,
                    destinations_count: destinationsCount
                };
                console.log('Matrix params:', matrixParams);
            } else {
                alert('Distance matrix requires origins_count and destinations_count to be greater than 0');
                document.getElementById('loadingSpinner').style.display = 'none';
                return;
            }
        }

        // Show loading
        document.getElementById('loadingSpinner').style.display = 'block';

        console.log('Sending request to:', this.apiBaseURL + '/calculate');
        console.log('API Requests:', apiRequests);
        console.log('Currency:', currency);

        // Make API call
        const requestBody = {
            api_requests: apiRequests,
            disable_new_customer_credit: disableNewCustomerCredit,
            disable_free_tier: disableFreeTier,
            currency: currency
        };
        
        // Add matrix_params if present
        if (matrixParams) {
            requestBody.matrix_params = matrixParams;
        }
        
        fetch(`${this.apiBaseURL}/calculate`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(requestBody)
        })
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            return response.json();
        })
        .then(data => {
            document.getElementById('loadingSpinner').style.display = 'none';
            console.log('Response received:', data);
            this.currentResults = data;
            this.displayResults(data);
            this.goToPage('results');
        })
        .catch(error => {
            document.getElementById('loadingSpinner').style.display = 'none';
            console.error('Error:', error);
            alert(`Failed to calculate costs.\n\nMake sure:\n1. Backend API is running on ${this.apiBaseURL}\n2. API server is not blocked by CORS\n\nError: ${error.message}`);
        });
    },

    // Display results
    displayResults(data) {
        if (!data.results || data.results.length === 0) {
            alert('No results returned');
            return;
        }

        // Display best value card
        const bestResult = data.results[0];
        const bestValueCard = document.getElementById('bestValueCard');
        // Use converted_cost when currency is provided and not USD
        const displayCost = (data.currency && data.currency !== 'USD' && bestResult.converted_cost) ? bestResult.converted_cost : bestResult.cost;
        bestValueCard.innerHTML = `
            <h3>Best Value: ${bestResult.name}</h3>
            <div class="price">${this.formatCurrency(displayCost, data.currency)}</div>
            <div class="unit">per month</div>
            ${data.currency && data.currency !== 'USD' ? `<div class="exchange-info">Exchange rate: 1 USD = ${Number(data.exchange_rate).toFixed(4)} ${data.currency}</div>` : ''}
            ${bestResult.notes ? `<p>${bestResult.notes}</p>` : ''}
        `;

        // Display all results
        const resultsGrid = document.getElementById('resultsGrid');
        resultsGrid.innerHTML = '';

        data.results.forEach((result, index) => {
            const card = document.createElement('div');
            card.className = 'result-card';
            if (index === 0) {
                card.classList.add('best-value');
            }

            let breakdownHTML = '';
            if (result.breakdown && Object.keys(result.breakdown).length > 0) {
                breakdownHTML = '<div class="breakdown"><strong>Breakdown by API:</strong>';
                for (const [apiType, breakdown] of Object.entries(result.breakdown)) {
                    const breakdownCost = data.currency === 'RUB' ? (breakdown.converted_cost || 0) : breakdown.cost;
                    const displayName = breakdown.display_name || this.formatAPIName(apiType);
                    const description = this.apiDescriptions[apiType];
                    const tooltipText = description ? description.description : '';
                    breakdownHTML += `
                        <div class="breakdown-item" title="${tooltipText}">
                            <span class="breakdown-api-name">${displayName}</span>
                            <span class="breakdown-cost">${this.formatCurrency(breakdownCost, data.currency)}</span>
                        </div>
                    `;
                }
                breakdownHTML += '</div>';
            }

            const resultCost = data.currency === 'RUB' ? result.converted_cost : result.cost;
            card.innerHTML = `
                ${index === 0 ? '<div style="color: var(--success); font-weight: bold; margin-bottom: 0.5rem;">BEST CHOICE</div>' : ''}
                <h4>${result.name}</h4>
                <div class="cost">${this.formatCurrency(resultCost, data.currency)}</div>
                <div style="color: var(--text-light); font-size: 0.9rem;">per month</div>
                ${result.notes ? `<div class="notes">${result.notes}</div>` : ''}
                ${breakdownHTML}
                <a href="${result.url}" target="_blank" class="url">View Pricing</a>
            `;

            resultsGrid.appendChild(card);
        });

        // Display summary table
        this.displaySummaryTable(data);
    },

    // Display summary table
    displaySummaryTable(data) {
        const summaryRows = document.getElementById('summaryRows');
        summaryRows.innerHTML = '';

        data.results.forEach(result => {
            const row = document.createElement('div');
            row.className = 'summary-row';

            const cost = (data.currency && data.currency !== 'USD' && result.converted_cost) ? result.converted_cost : result.cost;
            // Use per_request from backend when available (per single request)
            const perRequestRaw = result.per_request !== undefined ? Number(result.per_request) : null;

            let perRequestDisplay = '0.00';
            if (perRequestRaw !== null && !isNaN(perRequestRaw)) {
                if (data.currency === 'RUB') {
                    perRequestDisplay = 'руб' + perRequestRaw.toFixed(2);
                } else if (data.currency === 'EUR') {
                    perRequestDisplay = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'EUR', minimumFractionDigits: 2 }).format(perRequestRaw);
                } else {
                    perRequestDisplay = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', minimumFractionDigits: 2 }).format(perRequestRaw);
                }
            } else {
                perRequestDisplay = '-';
            }

            row.innerHTML = `
                <div>${result.name}</div>
                <div>${this.formatCurrency(cost, data.currency)}</div>
                <div>${perRequestDisplay}</div>
            `;

            summaryRows.appendChild(row);
        });
    },

    // Format currency
    formatCurrency(amount, currency = 'USD') {
        if (!currency || currency === 'USD') {
            return new Intl.NumberFormat('en-US', {
                style: 'currency',
                currency: 'USD',
                minimumFractionDigits: 2,
                maximumFractionDigits: 2
            }).format(amount || 0);
        }
        if (currency === 'RUB') {
            const formatted = new Intl.NumberFormat('ru-RU', {
                minimumFractionDigits: 2,
                maximumFractionDigits: 2
            }).format(amount || 0);
            return '₽ ' + formatted;
        }
        if (currency === 'EUR') {
            return new Intl.NumberFormat('en-IE', {
                style: 'currency',
                currency: 'EUR',
                minimumFractionDigits: 2,
                maximumFractionDigits: 2
            }).format(amount || 0);
        }

        // Fallback: format as plain number
        return (amount || 0).toFixed(2);
    }
};

// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    app.init();
});
