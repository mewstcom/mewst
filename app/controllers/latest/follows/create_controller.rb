# typed: true
# frozen_string_literal: true

class Latest::Follows::CreateController < Latest::ApplicationController
  include Authenticatable
  include Latest::FormErrorable

  def call
    form = Latest::FollowForm.new(
      viewer: current_viewer!,
      target_atname: params[:atname]
    )

    if form.invalid?
      return response_form_errors(resource_class: Latest::FollowFormErrorResource, errors: form.errors)
    end

    result = FollowProfileUseCase.new.call(
      viewer: form.viewer.not_nil!,
      target_profile: form.target_profile.not_nil!
    )

    resource = Latest::ProfileResource.new(profile: result.target_profile, viewer: current_viewer!)
    render(
      json: Latest::ProfileSerializer.new(resource),
      status: :created
    )
  end
end
