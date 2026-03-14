const app = {
    // Configuration - Backend API running on localhost:8080
    apiBaseURL: 'http://localhost:8080',
    currentResults: null,
    apiTypes: [],
    providers: [],

    // Initialize app
    init() {
        this.setupNavigation();
        this.loadProvidersAndInit();
        this.setupEventListeners();
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
                this.populateAPISelectors();
                this.populateProviders();
                console.log('Loaded ' + this.providers.length + ' providers');
                console.log('Loaded ' + this.apiTypes.length + ' API types');
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
            
            const apisList = provider.apis ? provider.apis.join(', ') : 'N/A';
            
            card.innerHTML += `
                <h3>${provider.name}</h3>
                <p class="desc">Visit official website</p>
                <ul>
                    <li>${apisList}</li>
                </ul>
                <a href="${provider.url}" target="_blank" style="color: var(--primary); text-decoration: none;">View Pricing</a>
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
            
            const label = document.createElement('label');
            label.textContent = this.formatAPIName(apiType);
            
            const checkbox = document.createElement('input');
            checkbox.type = 'checkbox';
            checkbox.value = apiType;
            checkbox.id = `api-${apiType}`;
            
            const input = document.createElement('input');
            input.type = 'number';
            input.min = '0';
            input.max = '10000000';
            input.value = '0';
            input.placeholder = 'Enter requests';
            input.id = `count-${apiType}`;
            input.disabled = true;
            
            // Enable/disable input based on checkbox
            checkbox.addEventListener('change', (e) => {
                input.disabled = !e.target.checked;
                if (e.target.checked) {
                    input.focus();
                } else {
                    input.value = '0';
                }
            });
            
            selector.appendChild(checkbox);
            selector.appendChild(label);
            selector.appendChild(input);
            selector.appendChild(document.createTextNode(' requests/month'));
            
            container.appendChild(selector);
        });
    },

    // Format API names for display
    formatAPIName(apiType) {
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

        // Show loading
        document.getElementById('loadingSpinner').style.display = 'block';

        console.log('Sending request to:', this.apiBaseURL + '/calculate');
        console.log('API Requests:', apiRequests);
        console.log('Currency:', currency);

        // Make API call
        fetch(`${this.apiBaseURL}/calculate`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                api_requests: apiRequests,
                disable_new_customer_credit: disableNewCustomerCredit,
                disable_free_tier: disableFreeTier,
                currency: currency
            })
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
        const cost = data.currency === 'RUB' ? bestResult.converted_cost : bestResult.cost;
        bestValueCard.innerHTML = `
            <h3>Best Value: ${bestResult.name}</h3>
            <div class="price">${this.formatCurrency(cost, data.currency)}</div>
            <div class="unit">per month</div>
            ${data.currency === 'RUB' ? `<div class="exchange-info">Exchange rate: 1 USD = ${data.exchange_rate.toFixed(2)} RUB</div>` : ''}
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
                breakdownHTML = '<div class="breakdown"><strong>Breakdown:</strong>';
                for (const [apiType, breakdown] of Object.entries(result.breakdown)) {
                    const breakdownCost = data.currency === 'RUB' ? (breakdown.converted_cost || 0) : breakdown.cost;
                    breakdownHTML += `
                        <div class="breakdown-item">
                            <span>${this.formatAPIName(apiType)}</span>
                            <span>${this.formatCurrency(breakdownCost, data.currency)}</span>
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

        const totalRequests = data.results[0].breakdown ? 
            Object.values(data.results[0].breakdown).reduce((sum, item) => sum + item.requests, 0) : 0;

        data.results.forEach(result => {
            const row = document.createElement('div');
            row.className = 'summary-row';
            
            const cost = data.currency === 'RUB' ? result.converted_cost : result.cost;
            const perRequest = totalRequests > 0 ? (cost / totalRequests * 1000).toFixed(6) : '0.00';
            
            const currencySymbol = data.currency === 'RUB' ? 'руб' : '$';

            row.innerHTML = `
                <div>${result.name}</div>
                <div>${this.formatCurrency(cost, data.currency)}</div>
                <div>${currencySymbol}${perRequest}</div>
            `;
            
            summaryRows.appendChild(row);
        });
    },

    // Format currency
    formatCurrency(amount, currency = 'USD') {
        if (currency === 'RUB') {
            return new Intl.NumberFormat('ru-RU', {
                style: 'currency',
                currency: 'RUB',
                minimumFractionDigits: 2,
                maximumFractionDigits: 2
            }).format(amount);
        }
        return new Intl.NumberFormat('en-US', {
            style: 'currency',
            currency: 'USD',
            minimumFractionDigits: 2,
            maximumFractionDigits: 2
        }).format(amount);
    }
};

// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    app.init();
});
