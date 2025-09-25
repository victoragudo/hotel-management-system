import http from 'k6/http';
import {check, sleep} from 'k6';
import {Rate, Trend} from 'k6/metrics';

export let errorRate = new Rate('errors');
export let responseTimeTrend = new Trend('response_time_custom');

export let options = {
    stages: [
        {duration: '1m', target: 10},   // Ramp up to 10 users over 1 minute
        {duration: '3m', target: 30},   // Ramp up to 30 users over 3 minutes
        {duration: '5m', target: 50},   // Sustained load of 50 users for 5 minutes
        {duration: '2m', target: 100},  // Spike to 100 users for 2 minutes
        {duration: '3m', target: 30},   // Scale down to 30 users for 3 minutes
        {duration: '1m', target: 0},    // Ramp down to 0 users over 1 minute
    ],

    thresholds: {
        http_req_duration: ['p(95)<500'],
        'http_req_duration{expected_response:true}': ['p(90)<200'],
        errors: ['rate<0.01'],
        response_time_custom: ['p(95)<400', 'p(99)<800'],
        http_req_failed: ['rate<0.01'],
        http_reqs: ['rate>20'],
    },

    noConnectionReuse: false,
    userAgent: 'k6-hotel-load-test/1.0.0',
};

const BASE_URL = 'http://localhost:8080';
const API_PATH = '/api/v1/hotels';

const HOTEL_IDS = [
    '1641879', '317597', '1202743', '1037179', '1154868', '1270324', '1305326', '1617655', '1975211', '2017823',
    '1503950', '1033299', '378772', '1563003', '1085875', '828917', '830417', '838887', '1702062', '1144294',
    '1738870', '898052', '906450', '906467', '2241195', '1244595', '1277032', '956026', '957111', '152896',
    '896868', '982911', '986491', '986622', '988544', '989315', '989544', '990223', '990341', '990370',
    '990490', '990609', '990629', '1259611', '991819', '992027', '992851', '993851', '994085', '994333',
    '994495', '994903', '995227', '995787', '996977', '1186578', '999444', '1000017', '1000051', '1198750',
    '1001100', '1001296', '1001402', '1002200', '1003142', '1004288', '1006404', '1006602', '1006810', '1006887',
    '1007101', '1007269', '1007466', '1011203', '1011644', '1011945', '1012047', '1012140', '1012944', '1023527',
    '1013529', '1013584', '1014383', '1015094', '1016591', '1016611', '1017019', '1017039', '1017044', '1018030',
    '1018130', '1018251', '1018402', '1018946', '1019473', '1020332', '1020335', '1020386', '1021856', '1022380'
];

function getRandomHotelId() {
    return HOTEL_IDS[Math.floor(Math.random() * HOTEL_IDS.length)];
}

export default function () {
    const hotelId = getRandomHotelId();
    const url = `${BASE_URL}${API_PATH}/${hotelId}`;

    const startTime = new Date();

    const response = http.get(url, {
        headers: {
            'Accept': 'application/json',
            'User-Agent': 'k6-hotel-load-test/1.0.0',
        },
        timeout: '30s',
    });

    const responseTime = new Date() - startTime;
    responseTimeTrend.add(responseTime);

    const result = check(response, {
        'status is 200': (r) => r.status === 200,
        'status is not 404': (r) => r.status !== 404,
        'status is not 500': (r) => r.status !== 500,

        'response time < 500ms': (r) => r.timings.duration < 500,
        'response time < 1000ms': (r) => r.timings.duration < 1000,

        'response has body': (r) => r.body && r.body.length > 0,
        'content-type is JSON': (r) => r.headers['Content-Type'] &&
            r.headers['Content-Type'].includes('application/json'),
    });

    if (response.status === 200) {
        check(response, {
            'response body contains hotel data': (r) => {
                try {
                    const jsonBody = JSON.parse(r.body);
                    return jsonBody &&
                        (jsonBody.id || jsonBody.hotel_id || jsonBody.data) &&
                        Object.keys(jsonBody).length > 0;
                } catch (e) {
                    return false;
                }
            },

            'response contains expected hotel fields': (r) => {
                try {
                    const jsonBody = JSON.parse(r.body);
                    return jsonBody && (
                        jsonBody.name ||
                        jsonBody.title ||
                        jsonBody.hotel_name ||
                        (jsonBody.data && jsonBody.data.name)
                    );
                } catch (e) {
                    return false;
                }
            }
        });
    }

    if (!result) {
        errorRate.add(1);
        console.log(`Request failed for hotel ID ${hotelId}: Status ${response.status}, Duration: ${response.timings.duration}ms`);
    } else if (response.status === 200) {
        errorRate.add(0);
    } else {
        errorRate.add(1);
    }

    if (response.timings.duration > 1000) {
        console.log(`Slow request for hotel ID ${hotelId}: ${response.timings.duration}ms`);
    }

    if (Math.random() < 0.05) { // 5% chance
        console.log(`Request success for hotel ID ${hotelId}: Status ${response.status}, Duration: ${Math.round(response.timings.duration)}ms`);
    }

    sleep(Math.random() * 2 + 0.5); // Random sleep between 0.5-2.5 seconds
}

export function setup() {
    console.log('Starting K6 Load Test for Hotel-by-ID endpoint');
    console.log(`Target URL: ${BASE_URL}${API_PATH}/{id}`);
    console.log(`Max concurrent users: 100`);
    console.log(`Total test duration: ~15 minutes`);
    console.log(`Testing with ${HOTEL_IDS.length} different hotel IDs`);

    const healthCheck = http.get(`${BASE_URL}/health`);
    if (healthCheck.status !== 200) {
        console.error(`Health check failed: ${healthCheck.status}`);
        throw new Error('Service is not healthy, aborting test');
    }

    console.log('Health check passed, starting load test...');
    return {startTime: new Date()};
}

export function teardown(data) {
    const endTime = new Date();
    const testDuration = (endTime - data.startTime) / 1000; // in seconds

    console.log('\n Load Test Completed!');
    console.log(`Total test duration: ${Math.round(testDuration)} seconds`);
    console.log('Check the summary above for detailed metrics');
    console.log('\n Key metrics to review:');
    console.log('   - http_req_duration (p95): Should be < 500ms');
    console.log('   - http_req_failed rate: Should be < 1%');
    console.log('   - error rate: Should be < 1%');
    console.log('   - http_reqs rate: Request throughput');
}