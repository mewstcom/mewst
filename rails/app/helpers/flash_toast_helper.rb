# typed: strict
# frozen_string_literal: true

module FlashToastHelper
  extend T::Sig

  sig { params(type: String, message: String).returns(String) }
  def dispatch_to_flash_toast(type:, message:)
    tag.div(
      data: {
        controller: "flash-toast-dispatch",
        flash_toast_dispatch_type_value: type,
        flash_toast_dispatch_message_html_value: message
      }
    )
  end
end
