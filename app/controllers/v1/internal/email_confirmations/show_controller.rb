# typed: true
# frozen_string_literal: true

class V1::Internal::EmailConfirmations::ShowController < V1::Internal::ApplicationController
  include InternalAuthenticatable

  def call
    email_confirmation = EmailConfirmation.find_by(id: params[:email_confirmation_id])

    unless email_confirmation
      resource = V1::ResponseErrorResource.new(code: V1::ResponseErrorCode::NotFound, message: "Not found")
      return render(
        json: V1::ResponseErrorSerializer.new([resource]),
        status: :not_found
      )
    end

    resource = V1::EmailConfirmationResource.new(email_confirmation:)
    render(
      json: V1::EmailConfirmationSerializer.new(resource)
    )
  end
end
