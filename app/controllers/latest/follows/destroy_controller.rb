# typed: true
# frozen_string_literal: true

class Latest::Follows::DestroyController < Latest::ApplicationController
  include Latest::FormErrorable

  def call
    form = Latest::UnfollowForm.new(
      viewer: current_profile.not_nil!,
      target_atname: params[:atname]
    )

    if form.invalid?
      return response_form_errors(resource_class: Latest::UnfollowFormErrorResource, errors: form.errors)
    end

    result = UnfollowProfileUseCase.new.call(
      viewer: form.viewer.not_nil!,
      target_profile: form.target_profile.not_nil!
    )

    resource = Latest::ProfileResource.new(profile: result.target_profile, viewer: current_profile.not_nil!)
    render(
      json: Latest::ProfileSerializer.new(resource),
      status: :ok
    )
  end
end
