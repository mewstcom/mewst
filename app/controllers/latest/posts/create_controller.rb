# typed: true
# frozen_string_literal: true

class Latest::Posts::CreateController < Latest::ApplicationController
  include Latest::Authenticatable
  include Latest::FormErrorable

  def call
    form = Latest::PostForm.new(
      viewer: current_viewer!,
      content: params[:content]
    )

    if form.invalid?
      return response_form_errors(resource_class: Latest::FormErrorResource, errors: form.errors)
    end

    result = CreatePostUseCase.new.call(
      viewer: form.viewer.not_nil!,
      content: form.content.not_nil!
    )

    resource = Latest::PostResource.new(post: result.post, viewer: current_viewer!)
    render(
      json: Latest::PostSerializer.new(resource),
      status: :created
    )
  end
end
