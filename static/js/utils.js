function modalToggle(selector, opened) {
	$(selector).toggleClass("open", opened);
}

$(document).ready(function() {
	/* Dropdown helper */
	$("body").on("click", ".dropdown", function(evt) {
		var container = $(".dropdown-container");

		if(!container.is(evt.target) && container.has(evt.target).length === 0) {
   			evt.preventDefault();

			var opened = $(evt.currentTarget).hasClass("open");
			$(".dropdown").removeClass("open"); /* close others */
			$(evt.currentTarget).toggleClass("open", !opened);
		}
	});

	$("body").click(function(evt) {
		var dropdown = $(".dropdown");

		if(!dropdown.is(evt.target) && dropdown.has(evt.target).length === 0) {
			dropdown.removeClass("open");
		}
	});

	/* Modal helper */
	$("a.modal-close").click(function(evt) {
   		evt.preventDefault();
		$(evt.target).closest(".modal").removeClass("open");
	});

	$(".modal").click(function(evt) {
		var body = $(".modal-content");

		if(!body.is(evt.target) && body.has(evt.target).length === 0) {
			$(evt.target).closest(".modal").removeClass("open");
		}
	});
});
