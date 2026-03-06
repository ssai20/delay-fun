let pollInterval;

document.addEventListener('DOMContentLoaded', function() {
    loadMeshTypes();
    document.getElementById('calculateBtn').addEventListener('click', startCalculation);
});

async function loadMeshTypes() {
    try {
        const response = await fetch('/api/mesh-types');
        const types = await response.json();

        const select = document.getElementById('meshType');
        types.forEach(type => {
            const option = document.createElement('option');
            option.value = type.id;
            option.textContent = type.name;
            select.appendChild(option);
        });
    } catch (error) {
        console.error('Error loading mesh types:', error);
    }
}

async function startCalculation() {
    // Показать статус
    const statusDiv = document.getElementById('status');
    statusDiv.style.display = 'block';
    statusDiv.className = 'status processing';
    statusDiv.textContent = '⏳ Расчет запущен, ожидайте...';

    // Скрыть предыдущие результаты
    document.getElementById('results').style.display = 'none';

    // Отключить кнопку
    document.getElementById('calculateBtn').disabled = true;

    // Получаем значения и заменяем запятые на точки
    const epsilonStart = document.getElementById('epsilonStart').value.replace(',', '.');
    const epsilonMin = document.getElementById('epsilonMin').value.replace(',', '.');
    const nStart = document.getElementById('nStart').value;
    const nMax = document.getElementById('nMax').value;
    const delta = document.getElementById('delta').value.replace(',', '.');

    const requestData = {
        epsilon_start: parseFloat(epsilonStart),
        epsilon_min: parseFloat(epsilonMin),
        n_start: parseInt(nStart),
        n_max: parseInt(nMax),
        delta: parseFloat(delta),
        mesh_type: document.getElementById('meshType').value
    };

    console.log('Sending request:', requestData); // Для отладки

    try {
        const response = await fetch('/api/calculate', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestData)
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const data = await response.json();
        console.log('Response:', data);

        if (data.status === 'processing') {
            startPolling(data.job_id);
        } else {
            showError('Неожиданный ответ от сервера');
        }
    } catch (error) {
        console.error('Error:', error);
        showError('Ошибка при запуске расчета: ' + error.message);
        document.getElementById('calculateBtn').disabled = false;
    }
}

function startPolling(jobId) {
    if (pollInterval) {
        clearInterval(pollInterval);
    }

    pollInterval = setInterval(async () => {
        try {
            const response = await fetch(`/api/status/${jobId}`);
            const data = await response.json();

            if (data.status === 'completed') {
                clearInterval(pollInterval);
                showResults(data);
            } else if (data.status === 'failed') {
                clearInterval(pollInterval);
                showError(data.error || 'Неизвестная ошибка');
            }
        } catch (error) {
            console.error('Polling error:', error);
        }
    }, 2000); // Проверяем каждые 2 секунды
}

function showResults(data) {
    document.getElementById('status').style.display = 'none';
    document.getElementById('calculateBtn').disabled = false;

    // Показать результаты
    const resultsDiv = document.getElementById('results');
    resultsDiv.style.display = 'block';

    // Ссылка на PDF
    document.getElementById('pdfLink').href = data.pdf_url;

    // Отобразить таблицы
    displayTable('classicTable', data.classic, 'Классическая схема');
    displayTable('modifiedTable', data.modified, 'Модифицированная схема');

    // Построить график
    createChart(data.classic, data.modified);
}

function displayTable(containerId, data, title) {
    const container = document.getElementById(containerId);

    let html = '<table>';
    html += '<tr><th>ε \\ N</th><th>128</th><th>256</th><th>512</th><th>1024</th><th>2048</th></tr>';

    const epsilons = ['1', '10⁻¹', '10⁻²', '10⁻³', '10⁻⁴', '10⁻⁵', '10⁻⁶', '10⁻⁷', '10⁻⁸'];

    for (let i = 0; i < data.length; i++) {
        html += '<tr>';
        html += `<td>${epsilons[i]}</td>`;
        for (let j = 0; j < data[i].length; j++) {
            html += `<td>${data[i][j]}</td>`;
        }
        html += '</tr>';
    }

    html += '</table>';
    container.innerHTML = html;
}

function createChart(classicData, modifiedData) {
    const ctx = document.getElementById('errorChart').getContext('2d');

    // Подготовка данных для графика (возьмем последний ε)
    const lastEpsilon = classicData[classicData.length - 1];
    const nValues = [128, 256, 512, 1024, 2048];

    // Уничтожить предыдущий график, если есть
    if (window.myChart) {
        window.myChart.destroy();
    }

    window.myChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: nValues,
            datasets: [
                {
                    label: 'Классическая схема',
                    data: lastEpsilon.map(val => parseFloat(val)),
                    borderColor: 'rgb(75, 192, 192)',
                    tension: 0.1
                },
                {
                    label: 'Модифицированная схема',
                    data: modifiedData[modifiedData.length - 1].map(val => parseFloat(val)),
                    borderColor: 'rgb(255, 99, 132)',
                    tension: 0.1
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                x: {
                    type: 'logarithmic',
                    title: {
                        display: true,
                        text: 'N'
                    }
                },
                y: {
                    type: 'logarithmic',
                    title: {
                        display: true,
                        text: 'Ошибка'
                    }
                }
            },
            plugins: {
                title: {
                    display: true,
                    text: 'Сходимость метода (ε = 10⁻⁸)'
                }
            }
        }
    });
}

function showError(message) {
    const statusDiv = document.getElementById('status');
    statusDiv.style.display = 'block';
    statusDiv.className = 'status error';
    statusDiv.textContent = '❌ ' + message;

    document.getElementById('calculateBtn').disabled = false;
}