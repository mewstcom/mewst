# typed: true
# frozen_string_literal: true

class V1::Posts::CreateController < V1::ApplicationController
  include ControllerConcerns::PublicAuthenticatable
  include ControllerConcerns::V1::FormErrorable

  def call
    form = V1::PostForm.new(
      viewer: current_viewer!,
      content: params[:content]
    )

    if form.invalid?
      return response_form_errors(resource_class: V1::FormErrorResource, errors: form.errors)
    end

    result = CreatePostUseCase.new.call(
      viewer: form.viewer.not_nil!,
      content: form.content.not_nil!
    )

    resource = V1::PostResource.new(post: result.post, viewer: current_viewer!)
    render(
      json: V1::PostSerializer.new(resource),
      status: :created
    )
  end
end
