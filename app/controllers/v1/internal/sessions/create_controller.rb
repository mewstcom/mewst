# typed: true
# frozen_string_literal: true

class V1::Internal::Sessions::CreateController < V1::Internal::ApplicationController
  include InternalAuthenticatable
  include V1::FormErrorable

  def call
    form = V1::SessionForm.new(
      email: params[:email],
      password: params[:password]
    )

    if form.invalid?
      return response_form_errors(resource_class: V1::SessionFormErrorResource, errors: form.errors)
    end

    result = CreateSessionUseCase.new.call(user: form.user.not_nil!)

    resource = V1::AccountResource.new(
      oauth_access_token: result.oauth_access_token,
      profile: result.profile,
      user: result.user
    )
    render(
      json: V1::AccountSerializer.new(resource),
      status: :created
    )
  end
end
