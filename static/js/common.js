function formatDurationLong(duration) {
	var seconds = duration % 60;
	var minutes = Math.floor(duration / 60);
	var hours   = Math.floor(minutes  / 60);
	var ret = "";

	if (hours > 0) {
		ret += hours
		ret += "h ";
	}

	if (minutes > 0) {
		ret += minutes
		ret += "m ";
	}

	ret += seconds
	ret += "s";

	return ret;
}

function formatTime(time) {
	var seconds = time.getSeconds();
	var minutes = time.getMinutes();
	var hours   = time.getHours();
	var ret = "";

	if (hours < 10) {
		ret += "0";
	}
	ret += hours
	ret += ":";

	if (minutes < 10) {
		ret += "0";
	}
	ret += minutes
	ret += ":";

	if (seconds < 10) {
		ret += "0";
	}
	ret += seconds

	return ret;
}

var serverUnixTime = $("#server-time").data("unixtime");

function updateTime() {
	serverUnixTime++;
	renderTime();
}

function renderTime() {
	var serverTime = new Date(serverUnixTime * 1000);
	var serverTimeStr = formatTime(serverTime);
	$("#server-time").text(serverTimeStr);
	$(".server-time").text(serverTimeStr);

	$(".expiration").each(function(i) {
		var target = $(this);

		var countdown = target.data("unixtime") - serverUnixTime;
		if (countdown < 0) {
			countdown = 0;
		}

		target.text(formatDurationLong(countdown));
	});
}

$(function() {
	renderTime();
	setInterval(updateTime, 1000);
});
