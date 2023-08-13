# typed: true
# frozen_string_literal: true

class Internal::Sessions::CreateController < Internal::ApplicationController
  def call
    form = Internal::SessionForm.new(
      email: params[:email],
      password: params[:password]
    )

    if form.invalid?
      resources = Internal::FormErrorResource.build_from_errors(errors: form.errors)
      return render(
        json: Internal::ResponseErrorSerializer.new(resources),
        status: :unprocessable_entity
      )
    end

    input = CreateSessionService::Input.from_internal_form(form:)
    result = ApplicationRecord.transaction do
      CreateSessionService.new.call(input:)
    end

    resource = Internal::AccountResource.new(
      oauth_access_token: result.oauth_access_token,
      profile: result.profile,
      user: result.user
    )
    render(
      json: Internal::AccountSerializer.new(resource),
      status: :created
    )
  end
end
