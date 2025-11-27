// ISS Tracker Application
let map;
let issMarker;
let issIcon;
let pathLine;
let pathCoordinates = [];
const MAX_PATH_POINTS = 50;

// API URLs
const ISS_POSITION_URL = 'http://api.open-notify.org/iss-now.json';
const ISS_PASS_URL = 'http://api.open-notify.org/iss-pass.json';

// Initialize the map
function initMap() {
    map = L.map('map', {
        center: [0, 0],
        zoom: 2,
        zoomControl: true,
        worldCopyJump: true
    });

    // Add tile layer
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '© OpenStreetMap contributors',
        maxZoom: 18
    }).addTo(map);

    // Create custom ISS icon
    issIcon = L.divIcon({
        className: 'iss-icon',
        html: '<div style="font-size: 32px; text-shadow: 2px 2px 4px rgba(0,0,0,0.8);">🛰️</div>',
        iconSize: [32, 32],
        iconAnchor: [16, 16]
    });

    // Initialize path line
    pathLine = L.polyline([], {
        color: '#64b5f6',
        weight: 2,
        opacity: 0.6,
        smoothFactor: 1
    }).addTo(map);
}

// Fetch ISS position
async function fetchISSPosition() {
    try {
        const response = await fetch(ISS_POSITION_URL);
        const data = await response.json();
        
        if (data.message === 'success') {
            const { latitude, longitude } = data.iss_position;
            const timestamp = data.timestamp;
            
            updateISSPosition(parseFloat(latitude), parseFloat(longitude), timestamp);
        }
    } catch (error) {
        console.error('Error fetching ISS position:', error);
        updateStatus('Error fetching data', false);
    }
}

// Update ISS position on map
function updateISSPosition(lat, lon, timestamp) {
    // Update or create marker
    if (issMarker) {
        issMarker.setLatLng([lat, lon]);
    } else {
        issMarker = L.marker([lat, lon], { icon: issIcon }).addTo(map);
        issMarker.bindPopup('<b>International Space Station</b><br>Current Location');
        map.setView([lat, lon], 3);
    }

    // Update path
    pathCoordinates.push([lat, lon]);
    if (pathCoordinates.length > MAX_PATH_POINTS) {
        pathCoordinates.shift();
    }
    pathLine.setLatLngs(pathCoordinates);

    // Update info panel
    document.getElementById('latitude').textContent = lat.toFixed(4) + '°';
    document.getElementById('longitude').textContent = lon.toFixed(4) + '°';
    
    // Update timestamp
    const date = new Date(timestamp * 1000);
    document.getElementById('lastUpdate').textContent = `Last updated: ${date.toLocaleTimeString()}`;
    updateStatus('Live', true);
}

// Update status indicator
function updateStatus(text, isOnline) {
    const statusElement = document.getElementById('status');
    if (isOnline) {
        statusElement.style.color = '#4caf50';
    } else {
        statusElement.style.color = '#ff6b6b';
    }
}

// Fetch visible passes for user location
async function fetchVisiblePasses(lat, lon) {
    try {
        const response = await fetch(`${ISS_PASS_URL}?lat=${lat}&lon=${lon}&n=5`);
        const data = await response.json();
        
        if (data.message === 'success') {
            displayPasses(data.response);
        }
    } catch (error) {
        console.error('Error fetching passes:', error);
        displayPassError('Unable to fetch pass times. Please try again.');
    }
}

// Display visible passes
function displayPasses(passes) {
    const passesContainer = document.getElementById('passes');
    
    if (passes.length === 0) {
        passesContainer.innerHTML = '<p class="info-text">No passes found for your location.</p>';
        return;
    }
    
    passesContainer.innerHTML = '';
    
    passes.forEach(pass => {
        const passDiv = document.createElement('div');
        passDiv.className = 'pass-item';
        
        const riseTime = new Date(pass.risetime * 1000);
        const duration = Math.round(pass.duration / 60);
        
        passDiv.innerHTML = `
            <div class="pass-time">${riseTime.toLocaleString()}</div>
            <div class="pass-duration">Duration: ${duration} minutes</div>
        `;
        
        passesContainer.appendChild(passDiv);
    });
}

// Display pass error
function displayPassError(message) {
    const passesContainer = document.getElementById('passes');
    passesContainer.innerHTML = `<p class="error">${message}</p>`;
}

// Get user location and fetch passes
function getUserLocationAndPasses() {
    const button = document.getElementById('getPassesBtn');
    button.disabled = true;
    button.textContent = 'Getting location...';
    
    if ('geolocation' in navigator) {
        navigator.geolocation.getCurrentPosition(
            (position) => {
                const { latitude, longitude } = position.coords;
                
                // Add user location marker
                L.marker([latitude, longitude], {
                    icon: L.divIcon({
                        className: 'user-icon',
                        html: '<div style="font-size: 24px;">📍</div>',
                        iconSize: [24, 24],
                        iconAnchor: [12, 12]
                    })
                }).addTo(map).bindPopup('<b>Your Location</b>');
                
                fetchVisiblePasses(latitude, longitude);
                button.textContent = 'Refresh Passes';
                button.disabled = false;
            },
            (error) => {
                console.error('Geolocation error:', error);
                displayPassError('Unable to get your location. Please enable location services.');
                button.textContent = 'Try Again';
                button.disabled = false;
            }
        );
    } else {
        displayPassError('Geolocation is not supported by your browser.');
        button.textContent = 'Not Available';
    }
}

// Initialize app
document.addEventListener('DOMContentLoaded', () => {
    initMap();
    fetchISSPosition();
    
    // Update position every 5 seconds
    setInterval(fetchISSPosition, 5000);
    
    // Add event listener for passes button
    document.getElementById('getPassesBtn').addEventListener('click', getUserLocationAndPasses);
});
