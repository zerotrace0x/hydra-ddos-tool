document.addEventListener('DOMContentLoaded', () => {
    const startAttackButton = document.getElementById('startAttack');
    const stopAttackButton = document.getElementById('stopAttack');
    const statusDiv = document.getElementById('status');
    const urlInput = document.getElementById('url');
    const workersInput = document.getElementById('workers');
    const postSizeInput = document.getElementById('post-size');

    const apiKey = process.env.API_KEY || 'your_secret_api_key'; // Replace with your actual API key
    const apiUrl = 'http://localhost:8080/api/attack';
    const apiStatusUrl = 'http://localhost:8080/api/status';
    let loading = false;

    const validateInput = () => {
        let isValid = true;

        // Clear previous error messages
        document.querySelectorAll('.error-message').forEach(e => e.remove());

        const url = urlInput.value;
        const workers = parseInt(workersInput.value, 10);
        const postSize = parseInt(postSizeInput.value, 10);

        if (!url.startsWith('http://') && !url.startsWith('https://')) {
            isValid = false;
            const errorElement = document.createElement('div');
            errorElement.classList.add('error-message');
            errorElement.textContent = 'URL must start with http:// or https://';
            urlInput.parentNode.insertBefore(errorElement, urlInput.nextSibling);
        }

        if (isNaN(workers) || workers <= 0) {
            isValid = false;
            const errorElement = document.createElement('div');
            errorElement.classList.add('error-message');
            errorElement.textContent = 'Workers must be a number greater than 0';
            workersInput.parentNode.insertBefore(errorElement, workersInput.nextSibling);
        }

        if (isNaN(postSize) || postSize <= 0) {
            isValid = false;
            const errorElement = document.createElement('div');
            errorElement.classList.add('error-message');
            errorElement.textContent = 'Post Size must be a number greater than 0';
            postSizeInput.parentNode.insertBefore(errorElement, postSizeInput.nextSibling);
        }

        return isValid;
    };

    const setLoading = (isLoading) => {
        loading = isLoading;
        startAttackButton.disabled = isLoading;
        stopAttackButton.disabled = isLoading;
        if (isLoading) {
            statusDiv.textContent = 'Loading...';
        }
    };

    const clearStatus = () => {
        statusDiv.textContent = '';
    };

    const displayError = (message) => {
        statusDiv.textContent = `Error: ${message}`;
    };

    const startAttack = () => {
        if (!validateInput()) {
            return;
        }

        const url = urlInput.value;
        const workers = parseInt(workersInput.value, 10);
        const postSize = parseInt(postSizeInput.value, 10);

        setLoading(true);
        clearStatus();
        statusDiv.textContent = 'Loading...'; // Show loading message

        fetch(apiUrl, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-API-Key': apiKey,
            },
            body: JSON.stringify({ url, workers, post_size: postSize }),
        })
            .then(response => {
                setLoading(false);
                if (response.status === 202) {
                    statusDiv.textContent = 'Attack started successfully.';
                } else {
                    displayError('Failed to start attack.');
                }
            })
            .catch(() => {
                setLoading(false);
                displayError('Failed to start attack.');
            });
    };

    const stopAttack = () => {
        setLoading(true);
        statusDiv.textContent = 'Loading...'; // Show loading message
        fetch(apiUrl, {
            method: 'DELETE',
            headers: {
                'X-API-Key': apiKey,
            },
        })
            .then(response => {
                setLoading(false);
                if (response.status === 200) {
                    statusDiv.textContent = 'Attack stopped successfully.';
                } else {
                    displayError('Failed to stop attack.');
                }
            })
            .catch(() => {
                setLoading(false);
                displayError('Failed to stop attack.');
            });
    };

    const getStatus = () => {
        fetch(apiStatusUrl, {
            headers: {
                'X-API-Key': apiKey,
            },
        })
            .then(response => {
                setLoading(false);
                return response.json()
            })
            .finally(() => {
                setLoading(false);
                getStatus(); // Refresh status after attempting to stop
            });
    });

    const getStatus = () => {
        fetch(apiStatusUrl)
            .then(response => response.json())
            .then(data => {
                if (data) {
                    let statusText = data.running ? 'Running' : 'Stopped';
                    let extraDetails = data.target_url ? `<p>Workers: ${data.workers}</p><p>Post Size: ${data.post_size}</p>` : '';
                    statusDiv.innerHTML = `
                        <p>Status: ${statusText}</p>
                        <p>Total Requests: ${data.total_requests}</p>
                        ${data.target_url ? `<p>Target URL: ${data.target_url}</p>` : ''}
                        ${extraDetails}
                    `;
                }
            })
            .catch(() => {
                setLoading(false);
                // Do not clear the status div, to be able to display the error messages
                // displayError('Failed to get status');
            })
            .finally(() => {
                if (!loading) {
                    setLoading(false);
                }
            });
    };

    startAttackButton.addEventListener('click', startAttack);
    stopAttackButton.addEventListener('click', stopAttack);

    setInterval(getStatus, 1000);
    getStatus();
});