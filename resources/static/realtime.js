

function StartRealtime(roomid, timestamp) {
    StartEpoch(timestamp);
    StartSSE(roomid);
    StartForm();
}

function StartForm() {
    $('#chat-message').focus();
    $('#chat-form').ajaxForm({
        beforeSubmit: function(arr, $form) {
            var messageValue = $('#chat-message').val().trim();
            if (!messageValue) {
                return false; // Prevent empty submissions
            }
            // Add loading state
            $('#chat-submit').prop('disabled', true).val('Sending...');
            return true;
        },
        success: function() {
            $('#chat-message').val('');
            $('#chat-message').focus();
            $('#chat-submit').prop('disabled', false).val('Send');
        },
        error: function(xhr) {
            console.error('Error sending message:', xhr.statusText);
            alert('Failed to send message. Please try again.');
            $('#chat-submit').prop('disabled', false).val('Send');
        }
    });
}

function StartEpoch(timestamp) {
    const windowSize = 60;
    const height = 200;
    const defaultData = histogram(windowSize, timestamp);

    window.heapChart = $('#heapChart').epoch({
        type: 'time.area',
        axes: ['bottom', 'left'],
        height: height,
        historySize: 10,
        data: [
            {values: defaultData},
            {values: defaultData}
        ]
    });

    window.mallocsChart = $('#mallocsChart').epoch({
        type: 'time.area',
        axes: ['bottom', 'left'],
        height: height,
        historySize: 10,
        data: [
            {values: defaultData},
            {values: defaultData}
        ]
    });

    window.messagesChart = $('#messagesChart').epoch({
        type: 'time.line',
        axes: ['bottom', 'left'],
        height: 240,
        historySize: 10,
        data: [
            {values: defaultData},
            {values: defaultData},
            {values: defaultData}
        ]
    });
}

function StartSSE(roomid) {
    if (!window.EventSource) {
        $('#chat').append('<tr class="danger"><td colspan="2">Your browser does not support Server-Sent Events. Please use a modern browser like Chrome, Firefox, or Safari.</td></tr>');
        return;
    }
    var source = new EventSource('/stream/'+roomid);
    source.addEventListener('message', newChatMessage, false);
    source.addEventListener('stats', stats, false);

    // Handle connection errors
    source.addEventListener('error', function(e) {
        if (e.target.readyState === EventSource.CLOSED) {
            $('#chat').append('<tr class="danger"><td colspan="2">Connection lost. Attempting to reconnect...</td></tr>');
        } else if (e.target.readyState === EventSource.CONNECTING) {
            $('#chat').append('<tr class="warning"><td colspan="2">Connecting to server...</td></tr>');
        }
    }, false);

    // Handle successful connection
    source.addEventListener('open', function() {
        $('#chat').append('<tr class="success"><td colspan="2">Connected to chat server.</td></tr>');
    }, false);
}

function stats(e) {
    var data = parseJSONStats(e.data);
    heapChart.push(data.heap);
    mallocsChart.push(data.mallocs);
    messagesChart.push(data.messages);
}

function parseJSONStats(e) {
    try {
        var data = JSON.parse(e);
    } catch (error) {
        console.error('Error parsing stats data:', error);
        return {heap: [], mallocs: [], messages: []};
    }
    var timestamp = data.timestamp;

    // Verify data properties exist before using them
    if (!data.timestamp || !data.HeapInuse || !data.StackInuse ||
        !data.Mallocs || !data.Frees ||
        !data.Connected || data.Inbound == null || data.Outbound == null) {
        console.error('Missing required properties in stats data');
        return {heap: [], mallocs: [], messages: []};
    }

    var heap = [
        {time: timestamp, y: data.HeapInuse},
        {time: timestamp, y: data.StackInuse}
    ];

    var mallocs = [
        {time: timestamp, y: data.Mallocs},
        {time: timestamp, y: data.Frees}
    ];
    var messages = [
        {time: timestamp, y: data.Connected},
        {time: timestamp, y: data.Inbound},
        {time: timestamp, y: data.Outbound}
    ];

    return {
        heap: heap,
        mallocs: mallocs,
        messages: messages
    }
}

function newChatMessage(e) {
    try {
        var data = JSON.parse(e.data);
    } catch (error) {
        console.error('Error parsing chat message:', error);
        return;
    }

    var nick = data.nick;
    var message = data.message;

    // Sanitize user input to prevent XSS
    nick = escapeHtml(nick);
    message = escapeHtml(message);

    var style = rowStyle(nick);
    var html = "<tr class=\""+style+"\"><td>"+nick+"</td><td>"+message+"</td></tr>";
    $('#chat').append(html);

    $("#chat-scroll").scrollTop($("#chat-scroll")[0].scrollHeight);
}

function histogram(windowSize, timestamp) {
    var entries = new Array(windowSize);
    for(var i = 0; i < windowSize; i++) {
        entries[i] = {time: (timestamp-windowSize+i-1), y:0};
    }
    return entries;
}

const entityMap = {
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': '&quot;',
    "'": '&#39;',
    "/": '&#x2F;'
};

function rowStyle(nick) {
    const classes = ['active', 'success', 'info', 'warning', 'danger'];
    const index = hashCode(nick) % 5;
    return classes[index];
}

function hashCode(s){
  return Math.abs(s.split("").reduce(function(a,b){a=((a<<5)-a)+b.charCodeAt(0);return a},0));
}

function escapeHtml(string) {
    return String(string).replace(/[&<>"'\/]/g, function (s) {
      return entityMap[s];
    });
}

window.StartRealtime = StartRealtime
