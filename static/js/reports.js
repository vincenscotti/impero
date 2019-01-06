$(function() {
	$("#select-btn").click(function(e) {
		checkBoxes = $("input[name=IDs]");
		checkBoxes.prop("checked", !checkBoxes.prop("checked"));
	});
});
