
$( document ).ready(function() {
    console.log( "ready!" );
});

$( '#linkAccount' ).on('click', function(event) {
	event.preventDefault();
	console.log('open '+authUri+' parent='+window.parent.location.href+' (location='+location.href+')');
	// databox pages don't pass through parameters to driver view
	window.parent.location.href = authUri;
	return false;
});

function reload() {
	console.log('reload!')
	location.reload()
}

function doSync(event) {
	event.preventDefault();
	console.log('sync ');
	$("#poll").prop('disabled', true);
	$.post('./ui/api/sync', {})
	.done(function () {
		console.log('Done sync');
		location.reload()
	})
	.fail(function() {
		console.log('Error requesting sync');
		alert("Error requesting sync");
		location.reload()
	});
	return false;
}

function configure(event) {
	var client_id = $("#client_id").val();
	var client_secret = $('#client_secret').val();
	if (!client_id) {
		alert("Client ID must be specified");
		return false;
	}
	if (needsClientSecret && !client_secret) {
		alert("Client Secret must be specified");
		return false;
	}
	console.log('Configure with ID='+client_id+' and secret='+client_secret);
	$("#configure").prop('disabled', true);
	$.ajax({
		type: "POST",
		url:'./ui/api/configure', 
		data: JSON.stringify({client_id:client_id, client_secret: client_secret}),
		dataType: "json"
	})
	.done(function () {
		console.log('Done configure');
		location.reload()
	})
	.fail(function() {
		console.log('Error configuring');
		alert("Sorry, there was a problem configuring the driver");
		location.reload()
	});
	return false;
}