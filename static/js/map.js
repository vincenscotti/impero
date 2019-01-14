powersupplylabels = { 0: "Alimentazione OK", 1: "In blackout", 2: "Adiacente ad un blackout" };

oldsize = "";
oldpadding = "";
target = undefined;

function isCompanyVisible(companyLabelObj) {
	return !companyLabelObj.hasClass("company-deselected");
}

function showCompany(companyLabelObj) {
	var cid = companyLabelObj.data("company-id");

	companyLabelObj.removeClass("company-deselected");
	$(".realnode[data-owner-id=" + cid + "]").removeClass("company-deselected");
}

function hideCompany(companyLabelObj) {
	var cid = companyLabelObj.data("company-id");

	companyLabelObj.addClass("company-deselected");
	$(".realnode[data-owner-id=" + cid + "]").addClass("company-deselected");
}

$(function() {
	$('#modal-node').on('hidden.bs.modal', function() {
		target.css("font-size", oldsize);
		target.css("padding", oldpadding);
	});

	$(".company-label").click(function(e) {
		var target = $(this);
		var wasHidden = !isCompanyVisible(target);

		if (wasHidden) {
			showCompany(target);
		} else {
			hideCompany(target);
		}
	});

	$("#company-selectall").click(function(e) {
		$(".company-label").each(function() {
			showCompany($(this));
		});
	});

	$("#company-hideall").click(function(e) {
		$(".company-label").each(function() {
			hideCompany($(this));
		});
	});

	$("td:not(.fakenode)").click(function(e) {
		target = $(this);
		var oldlink = $("#sel-owner-link").attr("href");

		if (target.data("owner-name") != undefined) {
			$("#sel-owner-name").text(target.data("owner-name"));
		} else {
			$("#sel-owner-name").text("");
		}
		$("#sel-owner-link").attr("href", oldlink.replace(/[^\/]*$/, target.data("owner-id")));
		$("#sel-x").text(target.data("x"));
		$(".sel-x").val(target.data("x"));
		$("#sel-y").text(target.data("y"));
		$(".sel-y").val(target.data("y"));
		$(".sel-owner-id").val(target.data("owner-id"));
		$("#sel-yield").text(target.data("yield") / 100);
		$("#sel-powersupply").text(powersupplylabels[target.data("powersupply")]);
		$("#sel-blackoutprob").text(Math.round(target.data("blackoutp") * 10000) / 100);
		$("#sel-buycost").text(target.data("buycost") / 100);
		if (target.data("investcost") != undefined) {
			$("#investbutton").show();
			$("#sel-investcost").text(target.data("investcost") / 100);
		} else {
			$("#investbutton").hide();
		}
		$("#sel-newyield").text(target.data("newyield") / 100);

		oldsize = target.css("font-size");
		oldpadding = target.css("padding");
		target.css("font-size", "x-large");
		target.css("padding", "10px");
	});
});
