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
        json: Panko::Response.new(
          errors: Panko::ArraySerializer.new(resources, each_serializer: Internal::ResponseErrorSerializer)
        ),
        status: :unprocessable_entity
      )
    end

    result = ApplicationRecord.transaction do
      input = CreateSessionService::Input.from_internal_form(form:)
      CreateSessionService.new.call(input:)
    end

    resource = Internal::AccountResource.new(
      oauth_access_token: result.oauth_access_token,
      profile: result.profile,
      user: result.user
    )
    render(
      json: Panko::Response.new(
        account: Internal::AccountSerializer.new.serialize(resource)
      ),
      status: :created
    )
  end
end
