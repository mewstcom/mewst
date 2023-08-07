# typed: true
# frozen_string_literal: true

class Internal::EmailConfirmations::ShowController < Internal::ApplicationController
  def call
    email_confirmation = EmailConfirmation.find_by(id: params[:email_confirmation_id])

    unless email_confirmation
      error = Internal::Resources::Base::Error.new(
        code: Internal::Resources::Base::ErrorCode::NotFound,
        message: "Not found"
      )
      return render(
        json: Internal::Resources::Error.new([error]),
        status: :not_found
      )
    end

    render(json: Internal::Resources::EmailConfirmation.new(email_confirmation))
  end
end
