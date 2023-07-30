# typed: true
# frozen_string_literal: true

class Internal::EmailConfirmations::ShowController < Internal::ApplicationController
  def call
    email_confirmation = EmailConfirmation.find_by(id: params[:email_confirmation_id])

    unless email_confirmation
      error = Resources::Internal::Base::Error.new(
        code: Resources::Internal::Base::ErrorCode::NotFound,
        message: "Not found"
      )
      return render(
        json: Resources::Internal::Error.new([error]),
        status: :not_found
      )
    end

    render(json: Resources::Internal::EmailConfirmation.new(email_confirmation))
  end
end
