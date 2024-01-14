# typed: true
# frozen_string_literal: true

class V1::Follows::DestroyController < V1::ApplicationController
  include ControllerConcerns::PublicAuthenticatable
  include ControllerConcerns::V1::FormErrorable

  def call
    form = V1::UnfollowForm.new(
      viewer: current_viewer!,
      target_atname: params[:atname]
    )

    if form.invalid?
      return response_form_errors(resource_class: V1::UnfollowFormErrorResource, errors: form.errors)
    end

    result = UnfollowProfileUseCase.new.call(
      viewer: form.viewer.not_nil!,
      target_profile: form.target_profile.not_nil!
    )

    resource = V1::ProfileResource.new(profile: result.target_profile, viewer: current_viewer!)
    render(
      json: V1::ProfileSerializer.new(resource),
      status: :ok
    )
  end
end
