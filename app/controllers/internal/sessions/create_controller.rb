# typed: true
# frozen_string_literal: true

class Internal::Sessions::CreateController < Internal::ApplicationController
  include Latest::FormErrorable

  def call
    form = Internal::SessionForm.new(
      email: params[:email],
      password: params[:password]
    )

    if form.invalid?
      return response_form_errors(resource_class: Internal::SessionFormErrorResource, errors: form.errors)
    end

    result = CreateSessionUseCase.new.call(user: form.user.not_nil!)

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
