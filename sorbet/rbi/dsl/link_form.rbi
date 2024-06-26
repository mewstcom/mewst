# typed: true

# DO NOT EDIT MANUALLY
# This is an autogenerated file for dynamic methods in `LinkForm`.
# Please instead update this file by running `bin/tapioca dsl LinkForm`.


class LinkForm
  sig { returns(T.nilable(::String)) }
  def canonical_url; end

  sig { params(value: T.nilable(::String)).returns(T.nilable(::String)) }
  def canonical_url=(value); end

  sig { returns(T.nilable(::String)) }
  def domain; end

  sig { params(value: T.nilable(::String)).returns(T.nilable(::String)) }
  def domain=(value); end

  sig { returns(T.nilable(::String)) }
  def image_url; end

  sig { params(value: T.nilable(::String)).returns(T.nilable(::String)) }
  def image_url=(value); end

  sig { returns(T.nilable(::String)) }
  def title; end

  sig { params(value: T.nilable(::String)).returns(T.nilable(::String)) }
  def title=(value); end
end
